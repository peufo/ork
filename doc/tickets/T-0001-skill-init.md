---
id: T-0001
titre: Skill /ork:init pour configurer un dépôt
statut: idee
priorite: normale
story:
plan_requis: false
zones: [skills/, agents/lead.md, README.md]
branche:
agent:
cree: 2026-10-29
maj: 2026-10-29
---

## Contexte
Constaté en démarrant ork sur Poj. Aujourd'hui, l'initialisation est une consigne
enfouie dans le prompt de `lead` (« si doc/ork.md n'existe pas, propose-le »). Elle n'est
déclenchée que par hasard, et les valeurs sont devinées.

Un skill dédié donnerait un point d'entrée explicite, qu'on peut relancer, et
allégerait le prompt de `lead`.

Ce qu'il devrait faire :
- détecter les commandes check, lint, tests et contrat, ainsi que la branche d'intégration ;
- repérer les dossiers de stories et de specs existants (Poj a `doc/stories/`, et non `doc/story/`) ;
- repérer les worktrees et skills d'intégration existants (Poj a un skill `integrate` qui
  fait un rebase sur `dev` sans commit de merge, ce qui contredit le `merge --no-ff` de `lead`) ;
- proposer, puis écrire après accord : `doc/ork.md`, `doc/tickets/`, `doc/tickets/done/` ;
- faire trancher le suivi git de `doc/tickets/` (commité ou ignoré).

## Critères de fin
- [ ] `skills/init/SKILL.md` existe et couvre la détection et la proposition ci-dessus
- [ ] `doc/ork.md` accepte des clés `methode_integration` (merge | rebase | skill:<nom>) et `dossier_stories`
- [ ] `agents/lead.md` lit ces clés au lieu de valeurs figées, et renvoie vers `/ork:init` si la config manque
- [ ] README mis à jour

## Plan

## Questions

## Journal
- 2026-10-29 — créé depuis les retours sur Poj
