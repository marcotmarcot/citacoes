from collections import Counter
import os
import pickle
import re
import sys

import spacy
from spacy import symbols


def main(text, source):
    with open(text) as fp:
        txt = fp.read()

    nlp = spacy.load('pt_core_news_sm')

    doc = nlp(txt, disable=['ner'])

    df = Counter()
    for token in doc:
        df.update(token.text.lower())

    for sent in doc.sents:
        sentence = process_sentence(df, source, sent)
        if sentence is not None:
            print(sentence)


def process_sentence(df, source, sent):
    answer = None
    tokens = []
    root_i = -1
    for token in sent:
        if token.dep_ == 'ROOT':
            root_i = token.i
            i = root_i - 1
            answer = []
            for left in reversed(list(token.lefts)):
                if left.i == i and left.dep_ in ['det', 'expl', 'advmod']:
                    i -= 1
                    tokens.pop()
                    answer.append(left.text)
                else:
                    break
            answer.append(token.text)
            answer = ' '.join(answer)
            tokens.append('__________')
        else:
            tokens.append(token.text)
    sentence, answer = post_process(join(tokens), answer)
    if is_ok(tokens, sentence, answer):
        return '"{} ({})","{}"'.format(sentence, source, answer)


def post_process(sentence, answer):
    sentence = re.sub(r'\s+', ' ', sentence).strip()
    # Add any - before or after ____ to the answer
    left, right = re.search(r'([^\s]+-)?_+(-[^\s]+)?', sentence).groups()
    if left:
        answer = left + answer
    if right:
        answer += right
    # Now replace the xxx-___-xxx pattern with ___
    sentence = re.sub(r'([^\s]+-)?(_+)(-[^\s]+)?', r'\2', sentence)
    return sentence, answer


def is_ok(tokens, sentence, answer):
    return answer and len(answer) > 3 \
        and ((sentence[0] != '_' and sentence[0] == sentence[0].upper()) or (sentence[0] == '_' and answer[0] == answer[0].upper())) \
        and (sentence[-1] in ('.', ';', '!', '?')) \
        and len(tokens) > 15


def join(tokens):
    sentence = ' '.join(tokens)
    sentence = sentence.replace(' .', '.')
    sentence = sentence.replace(' ;', ';')
    sentence = sentence.replace(' ,', ',')
    sentence = sentence.replace(' !', '!')
    sentence = sentence.replace(' ?', '?')
    sentence = sentence.replace(' - ', '-')
    return sentence


if __name__ == "__main__":
    main(sys.argv[1], sys.argv[2])
