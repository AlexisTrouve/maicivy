# 🚀 QuickStart - maicivy

## Démarrage rapide

### 1️⃣ Démarrer l'application complète

```bash
cd /mnt/c/Users/alexi/Documents/projects/maicivy
bash START.sh
```

**Le script va automatiquement :**
- ✅ Démarrer PostgreSQL (si pas déjà actif)
- ✅ Démarrer Redis (si pas déjà actif)
- ✅ Lancer le backend Go sur port 8080
- ✅ Lancer le frontend Next.js sur port 3000

**Temps de démarrage :** ~25 secondes

---

### 2️⃣ Accéder à l'application

Ouvre ton navigateur sur : **http://localhost:3000**

**Pages disponibles :**
- 🏠 Accueil : http://localhost:3000/
- 📄 CV Dynamique : http://localhost:3000/cv
- ✉️ Générateur de Lettres IA : http://localhost:3000/letters
- 📊 Analytics : http://localhost:3000/analytics

---

### 3️⃣ Arrêter l'application

```bash
bash STOP.sh
```

Arrête proprement le backend et le frontend (PostgreSQL et Redis restent actifs).

---

## 🧪 Tester les fonctionnalités

### ✅ CV Dynamique
1. Va sur http://localhost:3000/cv
2. Tu verras tes **7 expériences** professionnelles
3. Tes **20 compétences** (React, TypeScript, Go, PostgreSQL, etc.)
4. Tes **8 projets**

### ✅ Générateur de Lettres IA
1. Va sur http://localhost:3000/letters
2. Entre un nom d'entreprise (ex: "Google", "Microsoft", "OpenAI")
3. Clique sur "Générer"
4. Le système va :
   - 🔍 Scraper les infos de l'entreprise (Clearbit API)
   - 🤖 Utiliser ton vrai profil (AI Integration Developer, 8 ans d'exp)
   - ✨ Générer 2 lettres avec Claude AI :
     - Lettre de motivation professionnelle
     - Lettre "anti-motivation" humoristique
5. Compare les deux lettres côte à côte !

---

## 📊 Données chargées

**Profil actuel :**
- 👤 Nom : Alexi
- 💼 Poste : AI Integration Developer
- 🎯 Skills : React, TypeScript, Go, PostgreSQL, Docker, Redis, Next.js, Fiber
- 📅 Expérience : 8 ans (2016-2024)

**En base de données :**
- 7 expériences professionnelles
- 20 compétences techniques
- 8 projets portfolio

---

## 🔑 API Keys configurées

✅ `ANTHROPIC_API_KEY` (Claude 3.5 Sonnet)
✅ `OPENAI_API_KEY` (GPT-4 Turbo)

**La génération de lettres IA fonctionne !** 🎉
