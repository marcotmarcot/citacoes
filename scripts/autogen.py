from collections import Counter
import heapq
import os
import pickle
import re
import sys

import spacy
from spacy import symbols


def main(text, source):
    with open(text) as fp:
        txt = fp.read()

    nlp = spacy.load("pt_core_news_sm")

    doc = nlp(txt, disable=["ner"])

    df = Counter()
    for token in doc:
        df.update(token.text.lower())

    for sent in doc.sents:
        sentence = process_sentence(df, source, sent)
        if sentence is not None:
            print(sentence)


def process_sentence(df, source, sent):
    sentence, answer = post_process(sent, choose_token(df, sent))

    if is_ok(sentence, answer):
        return '"{} ({})","{}"'.format(sentence, source, answer)


def post_process(sent, answer):
    marker = "_" * 10

    start = answer[0].idx - sent.start_char
    end = (answer[-1].idx - sent.start_char) + len(answer[-1].text)
    new_sent = sent.text[:start] + marker + sent.text[end:]

    new_answer = sent.text[start:end]

    # Fix extra spaces
    new_sent = re.sub(r"\s+", " ", new_sent).strip()

    # Add any - before or after ____ to the answer
    left, right = re.search(r"(\w+-)?_+(-\w+)?", new_sent).groups()
    if left:
        new_answer = left + new_answer
    if right:
        new_answer += right

    # Now replace the xxx-___-xxx pattern with ___
    new_sent = re.sub(r"(\w+-)?(_+)(-\w+)?", r"\2", new_sent)

    return new_sent, new_answer


def choose_token(df, sent):
    candidates = []
    deps = ["nsubj", "nsubjpass", "dobj", "iobj", "ccomp", "xcomp"]
    for token in sent:
        if token.dep_ == "ROOT":
            candidates.append(expand(token))
        elif token.dep_ in deps and good_answer(token.text):
            candidates.append(expand(token))

    def dfsum(tokens):
        return sum(df.get(t, -1) for t in tokens)

    return heapq.nlargest(1, candidates, key=dfsum)[0]


def expand(token):
    deps = ["det", "expl", "advmod", "nummod", "compound"]
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


def good_answer(answer):
    return answer and len(answer) > 3


def is_ok(sentence, answer):
    return (
        good_answer(answer)
        and sentence[0] != "_"
        and sentence[0] == sentence[0].upper()
        and (sentence[0] not in (".", ";", "!", "?"))
        and (sentence[-1] in (".", ";", "!", "?"))
        and len(sentence) > 50
    )


if __name__ == "__main__":
    main(sys.argv[1], sys.argv[2])
