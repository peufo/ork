---
id: T-0001
title: Skill /ork:init pour configurer un dépôt
status: done
priority: normal
story:
plan_required: false
zones: [skills/, agents/lead.md, README.md]
branch:
agent:
created: 2026-10-29
updated: 2026-10-29
---

## Contexte
Constaté en démarrant ork sur Poj. Aujourd'hui, l'initialisation est une consigne
enfouie dans le prompt de `lead` (« si doc/ork.md n'existe pas, propose-le »). Elle n'est
déclenchée que par hasard, et les valeurs sont devinées.

Un skill dédié donnerait un point d'entrée explicite, qu'on peut relancer, et
allégerait le prompt de `lead`.

Ce qu'il devrait faire :
- détecter les commandes check, lint, tests et contrat, ainsi que la branche d'intégration ;
- repérer les dossiers existants qui ne suivent pas les conventions d'ork. Exemple : Poj a
  `doc/stories/`, alors qu'ork attend `doc/story/`. Pour chaque écart, demander à
  l'utilisateur ce qu'il préfère :
  - **renommer le dossier** (option recommandée, présentée en premier). Avant le renommage,
    lister tout ce qui le référence (docs, CLAUDE.md, code, scripts, configs) ; après le
    renommage, mettre ces références à jour ; enfin, relancer une recherche pour vérifier
    qu'il ne reste aucune référence à l'ancien chemin ;
  - **garder le dossier et ajouter une note** dans `doc/ork.md` (section Notes), qui indique
    l'emplacement réel ;
- repérer les worktrees et les skills d'intégration existants qui feraient doublon avec ork.
  Poj avait un skill `integrate`, absorbé depuis par le MODE INTÉGRATION. L'init devrait
  aussi vérifier que `CLAUDE.md` décrit les conventions dont les ouvriers ont besoin :
  préparation d'un worktree, fichiers générés, vérifications conditionnelles, style de commit ;
- proposer, puis écrire après accord : `doc/ork.md`, `doc/tickets/`, `doc/tickets/done/` ;
- faire trancher le suivi git de `doc/tickets/` (commité ou ignoré).

## Critères de fin
- [x] `skills/init/SKILL.md` existe et couvre la détection et la proposition ci-dessus
- [x] Pour un dossier hors convention, l'init pose la question « renommer ou noter », avec le renommage recommandé
- [x] Après un renommage, une recherche finale ne trouve plus aucune référence à l'ancien chemin (fichiers générés et `done/` exclus)
- [x] `agents/lead.md` renvoie vers `/ork:init` quand la config manque, au lieu de la proposer lui-même
- [x] README mis à jour

## Plan

## Questions

## Journal
- 2026-10-29 — créé depuis les retours sur Poj
- 2026-10-29 — dossiers hors convention : demander s'il faut renommer (recommandé, avec mise à jour des références) ou noter. La clé `dossier_stories` est abandonnée. Statut `pret`.
- 2026-10-29 — implémenté directement par le chef (phase de mise en place) : `skills/init/SKILL.md`, renvois depuis `lead` et `resume`, README. `plugin validate` OK. Pas encore testé en conditions réelles.
