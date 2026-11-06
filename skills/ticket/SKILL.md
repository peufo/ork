---
name: ticket
description: Capturer une remarque ou une capture d'écran en ticket, et le format d'un ticket ork. À charger par le chef dès qu'une remarque arrive ou qu'il crée un ticket.
---

Si tes instructions n'ont pas de section « Un sujet à la fois », tu n'es pas le chef : réponds seulement ``[WARN] à lancer depuis le chef, tape /ork`` et arrête-toi.

Le développeur est souvent en train de faire autre chose quand il t'envoie une remarque. Ne le retiens pas.

# Capturer une remarque

1. Si une capture d'écran est jointe et qu'elle n'existe pas encore sur disque, enregistre-la dans `.ork/tickets/captures/` sous un nom daté, pour que les ouvriers puissent la lire.
2. Vérifie qu'aucun ticket ouvert dans `.ork/tickets/` ne couvre déjà ce sujet. Si c'est le cas, ajoute la remarque à son contexte et à son journal au lieu de créer un doublon.
3. Si la remarque est claire et que tu sais quelles zones elle touche, rédige directement le ticket. Sinon, lance `Explore` (modèle `sonnet`) avec la remarque, le chemin de la capture et le format d'enquête ci-dessous. Ne garde que sa conclusion.
4. Crée le ticket, en `ready` si les critères de fin sont clairs, sinon en `idea`, et réponds par une ligne `[NEW]` : `` [NEW] `t29-slug` ready ``, ou `` [NEW] `t29-slug` idea : <la seule question> ``. Si la remarque suppose qu'un ticket encore ouvert soit fait d'abord, pose `depends_on`.
5. Ne lance pas le ticket, sauf s'il est `ready` et que les règles de parallélisme le permettent. Dans ce cas, lance-le.
6. Si le développeur estime que ça ne mérite pas de ticket, n'insiste pas.

# Format d'enquête

Demande à `Explore` de ne rien modifier, de lire la capture avec Read s'il y en a une, de ne pas proposer de correctif détaillé, et de rendre en 12 lignes maximum :

```
CONCLUSION: une ou deux phrases qui répondent à la question
ZONES: dossiers ou modules concernés, séparés par des virgules
KEY FILES: 1 à 5 chemins, avec un mot sur le rôle de chacun
CURRENT BEHAVIOR: ce que fait le code aujourd'hui, en une ou deux phrases
SUGGESTED TICKET: titre proposé, et 1 à 3 critères de fin vérifiables
PLAN REQUIRED: yes ou no, et pourquoi en une phrase
DOUBTS: ce que tu n'as pas pu établir (ou `none`)
```

# Format d'un ticket

```markdown
---
id: t12
title: Le bouton Enregistrer reste actif pendant l'envoi
status: ready
priority: normal
story: doc/story/edition-facture.md
depends_on: [t11]
plan_required: false
review:
model: sonnet
effort: medium
zones: [src/lib/components/facture, src/routes/factures]
branch:
agent:
paused:
created: 2026-10-25
updated: 2026-10-25
---

## Context
Ce que le développeur a remarqué ou demandé, et pourquoi c'est important.
Chemins des captures d'écran s'il y en a.

## Done criteria
- [ ] Comportement vérifiable 1
- [ ] Comportement vérifiable 2
- [ ] Les tests passent

## Plan
(rempli par l'ouvrier si plan_required, validé par le développeur avant implémentation)

## Questions
(questions ouvertes et réponses, datées)

## Review
(rempli par l'ouvrier à son DONE : résultat de la revue auto, puis les points à relire en cases à cocher, chacun en `path:line · N/10 · phrase`. Le chef ajoute `(auto)` à ceux qu'il coche d'office selon `confidence_threshold`)

## Log
- 2026-10-25 — créé depuis une remarque
```

Valeurs de `status` :
- `idea` : capturé, pas encore assez clair pour être lancé ;
- `ready` : critères de fin clairs, peut être lancé ;
- `planning` : un ouvrier prépare le plan ;
- `plan-review` : plan prêt, attend le développeur ;
- `in-progress` : un ouvrier implémente ;
- `blocked` : une question attend le développeur ;
- `in-review` : terminé par l'ouvrier, attend la relecture du développeur ;
- `done` : intégré, fichier déplacé dans `done/`.

**Qui écrit quoi.** L'ouvrier met à jour son propre ticket : `branch`, `updated`, `## Plan`, sa question dans `## Questions`, `## Review`, `## Log`, et le statut quand il rend `plan-review`, `blocked` ou `in-review`. Le chef crée le ticket et tient tout le reste : les réponses, les cases cochées, `agent`, les autres statuts, la clôture.

`priority` : `high`, `normal` ou `low`. `review` : vide, ou `required` pour forcer la relecture du développeur. Pose-le à la création si une règle de `## Review required when` s'applique au ticket. `zones` : les dossiers ou modules que le ticket touchera probablement, qui servent à décider du parallélisme. Dans un ticket, cite toujours un fichier par son chemin depuis la racine du dépôt.

`paused` : vide, ou la raison en quelques mots (`jusqu'à l'intégration de t34`). Le développeur met de côté un ticket `idea` ou `ready` : il quitte la file des sujets, le panneau le montre sous « En pause », et il ne part pas. Pose ce champ dès qu'il le demande, et vide-le quand il l'en sort, en le notant dans le journal.

`depends_on` : les `id` des tickets qui doivent être `done` avant que celui-ci parte. N'en pose une que si le ticket a besoin du code de l'autre, pas pour un simple ordre de préférence. Avant de l'écrire, vérifie que chaque `id` existe (dans `.ork/tickets/` ou `done/`) et qu'elle ne crée pas de cycle. Si un ticket est abandonné alors que d'autres en dépendent, demande au développeur quoi faire de ceux-ci : retirer la dépendance, ou les abandonner aussi.

Pour numéroter, prends le plus grand numéro parmi les `id` de `.ork/tickets/` et `.ork/tickets/done/` (compare les nombres, pas les noms de fichier : `t10` vient après `t9`), plus un.

$ARGUMENTS
