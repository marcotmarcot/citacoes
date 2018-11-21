from collections import Counter
import typing
from typing import Any, Optional, List, Tuple
import os
import pickle
import re
import sys

import spacy  # type: ignore
from spacy import symbols


def main(text: str, source: str):
    with open(text) as fp:
        txt = fp.read()

    nlp = spacy.load("pt_core_news_sm")
    doc = nlp(txt, disable=["ner"])

    # calculate word frequency
    wf: typing.Counter[str] = Counter(t.text for t in doc)

    for sent in doc.sents:
        sentence = process_sentence(wf, source, sent)
        if sentence is not None:
            print(sentence)


def process_sentence(wf: typing.Counter[str], source: str, sent: Any) -> Optional[str]:
    token = choose_token(wf, sent)
    sentence, answer = post_process(sent, token)

    if is_ok(sentence, answer):
        return '"{} ({})","{}"'.format(sentence, source, answer)

    return None


def post_process(sent: Any, answer: List[Any]) -> Tuple[str, str]:
    marker = "_" * 10

    start = answer[0].idx - sent.start_char
    end = (answer[-1].idx - sent.start_char) + len(answer[-1].text)
    new_sent = sent.text[:start] + marker + sent.text[end:]

    new_answer = sent.text[start:end]

    # Fix extra spaces
    new_sent = re.sub(r"\s+", " ", new_sent).strip()

    word = r"(?:\w|-)"
    regex = re.compile(r"({word}+-)?(_+)(-{word}+)?".format(word=word))
    # Add any - before or after ____ to the answer
    match = regex.search(new_sent)
    if match:
        left, _, right = match.groups()
        if left:
            new_answer = left + new_answer
        if right:
            new_answer += right

    # Now replace the xxx-___-xxx pattern with ___
    new_sent = regex.sub(r"\2", new_sent)

    return new_sent, new_answer


def choose_token(wf: typing.Counter[str], sent: Any) -> List[Any]:
    candidates: List[List[Any]] = []
    deps = ["ccomp", "csubj"]
    for token in sent:
        expanded = expand(token)
        if token.dep_ == "ROOT":
            candidates.append(expanded)
        elif token.dep_ in deps and good_answer(" ".join(t.text for t in expanded)):
            candidates.append(expanded)

    def wfsum(tokens):
        return sum(wf.get(t.text, -1) for t in tokens if len(t.text) > 2)

    return max(candidates, key=wfsum)


def expand(token: Any) -> List[Any]:
    deps = ["aux", "amod", "neg", "det", "expl", "advmod", "nummod", "compound"]
    expanded = [token]
    for left in reversed(list(token.lefts)):
        if left.i == (expanded[0].i - 1) and left.dep_ in deps:
            expanded.insert(0, left)
        else:
            break
    for right in token.rights:
        if right.i == (expanded[-1].i + 1) and right.dep_ in deps:
            expanded.append(right)
        else:
            break
    return expanded


def good_answer(answer: str) -> bool:
    return bool(answer and len(answer) > 3)


def is_ok(sentence: str, answer: str) -> bool:
    return (
        good_answer(answer)
        and sentence[0] != "_"
        and sentence[0] == sentence[0].upper()
        and (sentence[0] not in ".;!?:")
        and (sentence[-1] in ".;!?")
        and len(sentence) > 50
    )


if __name__ == "__main__":
    main(sys.argv[1], sys.argv[2])
