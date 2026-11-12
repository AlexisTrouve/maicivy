"""mesures.py — les cinq mesures de recette du gabarit de CV PDF (templates/cv/cv_base.html).

Usage : python mesures.py <dossier html> <dossier pdf> <wqy-microhei.ttf> <nom> [<nom>...]
  <nom> = préfixe produit par TestRecettePDF_CheminExport (CV_RECETTE_NOM, défaut « prod »).
  Les PDF doivent sortir de l'IMAGE Docker du backend (headless-shell 123 + ses polices) : un
  Chromium local rend avec d'autres polices et d'autres métriques, sa mesure ne dit rien de la prod.
  Dépendances : PyMuPDF, Pillow, fontTools, et pdftoppm (poppler) dans le PATH.

Pour chaque nom et chaque langue (fr, zh) :
 1. EXTRACTION — le texte des couches cachées est-il dans le texte extrait du PDF ? On lit les
    couches dans le HTML rendu (ce que le gabarit a réellement émis), puis on les cherche dans le
    texte PyMuPDF, espaces retirés des deux côtés (le PDF recoupe les lignes à sa façon).
    Polices embarquées listées : une police Type3 = texte non extractible (cas Noto CJK en CFF).
 2. INVISIBILITÉ — rendu avec / sans couches cachées (TestRecettePDF_CheminExport : visibility:hidden, même mise
    en page), rastérisé à 150 dpi par DEUX moteurs (MuPDF et poppler) : écart maximal par canal,
    à comparer au seuil 1/255. Deux moteurs parce que leur conversion de couleur diffère : un
    écart de 1 unité écrit dans le PDF peut sortir à 2 chez l'un et 1 chez l'autre.
 3. POIDS du PDF.  4. NOMBRE DE PAGES.
 5. TOFU (zh) — chaque idéogramme du texte visible est-il dans la table cmap de WenQuanYi (la
    police CJK de l'image Docker) ? Et dans le PDF, quels spans CJK sont rendus par une AUTRE
    police (= repli système Windows, qui masquerait un tofu de prod) ?
"""
import glob
import html as htmlmod
import os
import re
import subprocess
import sys
import tempfile

import fitz  # PyMuPDF
from PIL import Image, ImageChops
from fontTools.ttLib import TTFont

html_dir, pdf_dir, wqy_path, *noms = sys.argv[1:]
wqy_cmap = set(TTFont(wqy_path)['cmap'].getBestCmap())
cjk = re.compile(r'[\u3000-\u303f\u3400-\u9fff\uff00-\uffef]')


def sans_espaces(s):
    return re.sub(r'\s+', '', s)


def couches(source):
    """Texte des couches cachées, tel que le gabarit les a émises."""
    def blocs(cls, tag):
        return [htmlmod.unescape(m) for m in re.findall(rf'<{tag} class="{cls}">(.*?)</{tag}>', source, re.S)]
    return {
        'strip': blocs('header-stealth', 'span'),
        'stealth': blocs('stealth', 'div'),
        'ancre': blocs('ats-anchor', 'span'),
    }


def texte_visible(source):
    corps = re.sub(r'<style.*?</style>', '', source, flags=re.S)
    for cls in ('header-stealth', 'stealth', 'ats-anchor'):
        corps = re.sub(rf'<(span|div) class="{cls}">.*?</\1>', '', corps, flags=re.S)
    return htmlmod.unescape(re.sub(r'<[^>]+>', ' ', corps))


def pages_mupdf(pdf):
    for p in fitz.open(pdf):
        pm = p.get_pixmap(dpi=150, alpha=False)
        yield Image.frombytes('RGB', (pm.width, pm.height), pm.samples)


def pages_poppler(pdf):
    with tempfile.TemporaryDirectory() as tmp:
        subprocess.run(['pdftoppm', '-r', '150', '-png', pdf, os.path.join(tmp, 'p')], check=True)
        for f in sorted(glob.glob(os.path.join(tmp, 'p-*.png'))):
            yield Image.open(f).convert('RGB')


def ecart_max(pdf_a, pdf_b, moteur):
    """Écart maximal par canal entre deux rendus, et nombre de valeurs de canal qui diffèrent."""
    pages_a, pages_b = list(moteur(pdf_a)), list(moteur(pdf_b))
    if len(pages_a) != len(pages_b):
        return None, None
    pire, n = 0, 0
    for a, b in zip(pages_a, pages_b):
        d = ImageChops.difference(a, b)
        pire = max(pire, max(hi for _, hi in d.getextrema()))
        n += sum(c for v, c in enumerate(d.histogram()) if v % 256)
    return pire, n


for nom in noms:
    for lang in ('fr', 'zh'):
        base = f'{nom}-{lang}'
        source = open(os.path.join(html_dir, base + '.html'), encoding='utf-8').read()
        pdf_path = os.path.join(pdf_dir, base + '.pdf')
        doc = fitz.open(pdf_path)
        extrait = sans_espaces(''.join(p.get_text() for p in doc))
        print(f'== {base} ==')

        # 1. Extraction
        c = couches(source)
        strip = c['strip'][0] if c['strip'] else ''
        termes = [t.strip() for t in re.split(r'[,.:]\s', strip) if t.strip()]
        trouves = [t for t in termes if sans_espaces(t) in extrait]
        print(f'1. strip : {len(strip.encode())} octets (budget 780) ; strip entier présent d\'un seul tenant : '
              f'{sans_espaces(strip) in extrait} ; termes présents {len(trouves)}/{len(termes)}')
        manquants = [t for t in termes if t not in trouves]
        if manquants:
            print(f'   termes absents : {manquants}')
        entiers = sum(1 for s in c['stealth'] if sans_espaces(s) in extrait)
        # Couverture mot à mot : une couche peut sortir en deux morceaux (ordre de lecture du
        # PDF) sans rien perdre — une comparaison d'un seul tenant la compterait comme perdue.
        mots = [m for s in c['stealth'] for m in s.split()]
        mots_trouves = sum(1 for m in mots if m in extrait)
        print(f'   couches .stealth : {entiers}/{len(c["stealth"])} d’un seul tenant ; '
              f'mots retrouvés {mots_trouves}/{len(mots)} ({100 * mots_trouves / max(len(mots), 1):.1f} %)')
        print(f'   ancres ATS : {sum(1 for a in c["ancre"] if sans_espaces(a) in extrait)}/{len(c["ancre"])}')
        polices = sorted({(f[3], f[2]) for p in doc for f in p.get_fonts() if f[2] != 'Type3'})
        type3 = len({f[0] for p in doc for f in p.get_fonts() if f[2] == 'Type3'})
        print(f'   polices embarquées (nom, type) : {polices} ; polices Type3 : {type3}')

        # 2. Invisibilité
        for libelle, moteur in (('MuPDF', pages_mupdf), ('poppler', pages_poppler)):
            pire, n = ecart_max(pdf_path, os.path.join(pdf_dir, base + '-sans-stealth.pdf'), moteur)
            if pire is None:
                print(f'2. diff avec/sans couches ({libelle}) : NOMBRE DE PAGES DIFFÉRENT')
            else:
                print(f'2. diff avec/sans couches ({libelle}, 150 dpi) : écart max {pire}/255 sur {n} valeurs '
                      f'de canal → {"OK" if pire <= 1 else "ÉCHEC"} (seuil 1/255)')

        # 3. et 4.
        print(f'3. poids : {os.path.getsize(pdf_path)} octets ({os.path.getsize(pdf_path) / 1024:.0f} Ko)')
        print(f'4. pages : {doc.page_count}')

        # 5. Tofu
        ideos = sorted(set(cjk.findall(texte_visible(source))))
        hors_wqy = [ch for ch in ideos if ord(ch) not in wqy_cmap]
        brut = ''.join(p.get_text() for p in doc)
        absents_pdf = [ch for ch in ideos if ch not in brut]
        repli, cjk_type3 = set(), 0
        for p in doc:
            for bloc in p.get_text('dict')['blocks']:
                for ligne in bloc.get('lines', []):
                    for span in ligne['spans']:
                        if not cjk.search(span['text']):
                            continue
                        # Type3 = glyphes en gras SIMULÉ (WenQuanYi n'a pas de gras) : même police,
                        # dessinée en chemins. Tout autre nom = repli sur une police système.
                        if span['font'].startswith('Type3'):
                            cjk_type3 += 1
                        elif 'WenQuanYi' not in span['font']:
                            repli.add(span['font'])
        print(f'5. CJK : {len(ideos)} caractères distincts dans le texte visible ; absents de WenQuanYi : '
              f'{len(hors_wqy)} {hors_wqy[:20]} ; absents du texte extrait : {len(absents_pdf)} {absents_pdf[:20]} ; '
              f'spans CJK en Type3 (gras simulé) : {cjk_type3} ; polices système de repli : {sorted(repli) or "aucune"} ; '
              f'U+FFFD : {brut.count(chr(0xFFFD))}')
        print()
