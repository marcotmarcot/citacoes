from collections import Counter
from operator import itemgetter
import math
import os
import pickle
import re
import sys
from typing import Any, Dict, Optional, List, Tuple
import typing

import spacy  # type: ignore
from spacy import attrs, symbols
from spacy.tokens import Doc, Span, Token  # type: ignore

Span.set_extension("answers", default=[])
Span.set_extension("scored_answers", default=[])

# These are words that should be aggregated into the answer to make the phrase
# less specific.
EXPAND_DEPS = [
    "advmod",  # adverbial modifier. Eg.: A seca afeta -> *pouco* a produção de grãos
    "amod",  # adjectival modifier. Eg.: Desde o *último* <- dia 13, ...
    "aux",  # auxiliary (verb). Eg.: O mesmo não se *pode* <- dizer, ... *tenham* <- estudado, etc
    "compound",  # compound names. Eg.: Passei na *linha <- amarela*
    "fixed",  # another kind of compounding
    "flat",  # another kind of compounding
    "det",  # determinant. Eg.: *Meu* <- refrigerador não funciona
    "neg",  # negative. Eg. *Não* <- faça nada
    "nummod",  # numeric modifier. Eg.: fiz *30* <- episódios
]

# Only sentences with these types of roots should be allowed
ALLOWED_ROOT_POS = ["VERB", "NOUN", "PROPN"]

# Only answers with one of these root tokens are allowed
ALLOWED_ANSWER_POS = ["ADJ", "NOUN", "PROPN", "VERB"]

# Clauses are good candidates because they are more semantically contained.
ALLOWED_ANSWER_DEPS = [
    "ROOT",  # root of the sentence (usually a verb),
    "obj",  # object (usually a noun or proper noun)
    "obl",  # oblique nominal (unsure about this, usually noun)
    "nsubj",  # nominal subject (usually a noun or proper noun)
    "conj",  # conjunct (usually a advective or noun. Eg.: Ela é bonita e *simpática*)
]


def main(text: str, wordfreq: str, source: str):
    # setup
    with open(text) as fp:
        txt = fp.read()

    nlp = spacy.load("pt_core_news_sm")
    doc = nlp(txt, disable=["ner"])

    wf: Dict[str, int] = {}
    with open(wordfreq) as fp:
        for t in fp:
            if t:
                word, freq = t.split()
                wf[word] = int(freq)
    for norm, freq in doc.count_by(attrs.NORM).items():
        wf[doc.vocab[norm].norm_] = int(freq)

    def good_sentence(sent: Span) -> bool:
        tokens = [t for t in sent if t.text.strip()]
        return (
            tokens[0].is_title
            and sent.root.pos_ in ALLOWED_ROOT_POS
            and not tokens[0].is_punct
            and tokens[-1].text in ".;!"
            and len(sent.text) > 50
        )

    # Prefilter sentences
    sents = [sent for sent in doc.sents if good_sentence(sent)]

    annotate_answers(sents, wf)

    for sent in sents:
        answer, score = max(sent._.scored_answers, key=itemgetter(1), default=("", -1))

        if score >= 0:
            sentence, answer = post_process(sent, answer)
            print('"{} ({})","{}"'.format(sentence, source, answer))


def annotate_answers(sents: List[Span], wf: Dict[str, int]):
    for sent in sents:
        # Prefilter answers
        answers = [
            token
            for token in sent
            if token.dep_ in ALLOWED_ANSWER_DEPS and token.pos_ in ALLOWED_ANSWER_POS
        ]
        sent._.answers = [expand_answers(answer) for answer in answers]
        sent._.scored_answers = score_answers(sent, wf)


def expand_answers(token: Token) -> List[List[Token]]:
    expanded = [token]

    for left in reversed(list(token.lefts)):
        if left.i == (expanded[0].i - 1) and left.dep_ in EXPAND_DEPS:
            expanded.insert(0, left)
        else:
            break
    for right in token.rights:
        if right.i == (expanded[-1].i + 1) and right.dep_ in EXPAND_DEPS:
            expanded.append(right)
        else:
            break

    def fix_o_senhor_seu_deus(expanded):
        pre = ["o senhor"]
        pos = ["nosso deus", "teu deus", "seu deus"]
        answer = " ".join(t.text for t in expanded).lower()
        if answer in pos:
            nbors = [expanded[0].nbor(-2), expanded[0].nbor(-1)]
            if " ".join([w.text.lower() for w in nbors]) in pre:
                expanded = nbors + expanded
        if answer in pre:
            nbors = [expanded[-1].nbor(), expanded[-1].nbor(2)]
            if " ".join([w.text.lower() for w in nbors]) in pos:
                expanded = expanded + nbors
        return expanded

    def fix_hifens(expanded):
        try:
            while expanded[0].nbor(-1).text == "-":
                nbors = [expanded[0].nbor(-2), expanded[0].nbor(-1)]
                if (
                    expanded[0].sent == nbors[0].sent
                    and expanded[0].sent == nbors[1].sent
                ):
                    expanded = nbors + expanded
                else:
                    break
        except IndexError:
            pass

        try:
            while expanded[-1].nbor().text == "-":
                nbors = [expanded[-1].nbor(), expanded[-1].nbor(2)]
                if (
                    expanded[-1].sent == nbors[0].sent
                    and expanded[-1].sent == nbors[1].sent
                ):
                    expanded = expanded + nbors
                else:
                    break
        except IndexError:
            pass

        return expanded

    def fix_contractions(expanded):
        try:
            prev = expanded[0].nbor(-1)
        except IndexError:
            return expanded

        if expanded[0].sent == prev.sent and not prev.whitespace_ and prev.text in "ad":
            expanded = [prev] + expanded

        return expanded

    expanded = fix_o_senhor_seu_deus(expanded)
    expanded = fix_hifens(expanded)
    expanded = fix_contractions(expanded)

    return expanded


def score_answers(sent: Span, wf: Dict[str, int]) -> List[Tuple[Token, float]]:
    tf = Counter(t.norm for t in sent)

    def good_answer(answer: List[Token]) -> bool:
        text = " ".join(a.text for a in answer)
        return bool(answer and not answer[0].is_sent_start and len(text) > 3)

    def score(tokens):
        if not good_answer(tokens):
            return -1
        # filter likely stop words
        tokens = [t for t in tokens if len(t.norm_) > 2]
        return sum(math.log(wf.get(t.norm_, 1) + 1) / tf.get(t.norm, 1) for t in tokens)

    return [(answer, score(answer)) for answer in sent._.answers]


def post_process(sent: Any, answer: List[Token]) -> Tuple[str, str]:
    marker = "_" * 10

    start = answer[0].idx - sent.start_char
    end = (answer[-1].idx - sent.start_char) + len(answer[-1].text)
    new_sent = sent.text[:start] + marker + sent.text[end:]

    new_answer = sent.text[start:end]

    # Fix extra spaces
    new_sent = re.sub(r"\s+", " ", new_sent).strip()

    return new_sent, new_answer


if __name__ == "__main__":
    main(sys.argv[1], sys.argv[2], sys.argv[3])
