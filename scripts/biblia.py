import sys

import spacy


def good_sentence(tokens, sentence):
    return sentence[0] == sentence[0].upper() \
        and (sentence.endswith('.') or sentence.endswith(';')) \
        and 10 < len(tokens) < 40


def join(tokens):
    sentence = ' '.join(tokens)
    sentence = sentence.replace(' .', '.')
    sentence = sentence.replace(' ;', ';')
    sentence = sentence.replace(' ,', ',')
    sentence = sentence.replace(' !', '!')
    sentence = sentence.replace(' ?', '?')
    sentence = sentence.replace(' - ', '-')
    return sentence


def main(text, source):
    with open(text) as fp:
        deut = fp.read()

    nlp = spacy.load('pt_core_news_sm')

    doc = nlp(deut, disable=['ner'])

    for sent in doc.sents:
        answer = None
        tokens = []
        for token in sent:
            if token.dep_ == 'ROOT':
                answer = token.text.replace('\n', ' ').strip()
                tokens.append('__________')
            else:
                tokens.append(token.text.replace('\n', ' ').strip())
        if answer and len(answer) > 4:
            sentence = join(tokens)
            if good_sentence(tokens, sentence):
                print('"{} ({})","{}"'.format(sentence.strip(), source, answer))


main(sys.argv[1], sys.argv[2])
