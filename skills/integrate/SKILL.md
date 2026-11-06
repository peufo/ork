---
name: integrate
description: Intégrer une branche d'ouvrier ou un worktree sur la branche d'intégration, et reprendre un ouvrier injoignable. À charger par le chef avant toute intégration.
---

# Intégrer

La méthode est unique, et l'historique reste linéaire :
1. la branche est rebasée sur `integration_branch` et vérifiée **dans son worktree**, par un ouvrier ;
2. tu la poses ensuite par avance rapide (`merge --ff-only`).

`integration_branch` ne bouge que lorsque tout est vert.

Tu la déclenches toi-même après un `DONE` qui ne demande pas de relecture. Sinon, le développeur la déclenche par « intègre t12 », ou par « intègre <worktree ou branche> » pour un worktree qui ne vient pas d'un ticket. S'il y en a plusieurs : un à la fois, chacun rebasé sur la branche laissée par le précédent. Un échec n'arrête pas les suivants : laisse-le en place et note-le.

Deux règles ne se négocient pas, parce que rien ne se rattrape :
- **le dépôt principal doit être propre**. Une modification qui s'y trouve appartient au développeur et n'existe nulle part ailleurs. Ne la contourne jamais par un stash, un reset ou un checkout ;
- **jamais de push, jamais de force** : ni `--force`, ni `worktree remove --force`, ni `branch -D`. Un refus de git signale du travail non rapatrié. Regarde ce qui reste au lieu de forcer.

Toutes les commandes passent par `git -C <path>`.

1. **Repérer.** `git worktree list --porcelain` : le premier bloc est le dépôt principal. S'il est sur `review/*`, une relecture est ouverte : ne pose rien, note `pose en attente` dans le journal du ticket, et arrête-toi sans avertir. `ork:review` reprend cette pose à la fermeture. Sinon, il doit être sur `integration_branch`, et `git status --short` doit être vide, fichiers non suivis compris. Seuls `.ork/config.md`, `.ork/tickets/` et `.claude/launch.json`, que l'app peut créer seule, sont tolérés. Sinon, avertis en nommant la branche ou les fichiers, et arrête-toi.
2. **Vérifier qu'aucune session ne tourne** dans le worktree visé. Les indices :
   - un verrou (`locked`) dont le pid est vivant (`ps -p <pid>`) et n'est pas le tien ;
   - une session dans ce dossier selon `ListAgents` ;
   - des fichiers modifiés il y a quelques minutes.

   Si c'est le cas, demande avant d'aller plus loin.

   **Ton propre verrou.** Claude Code verrouille le worktree d'un ouvrier avec le pid de la session qui l'a lancé, c'est-à-dire toi, et ne le lève qu'à la fin de ta session. Ton pid est le parent de ton shell : `ps -o ppid= -p $$`. Un verrou à ton pid ne signale donc pas une session dans le worktree. Ce qui compte, c'est que ton ouvrier ait rendu son rapport et ne travaille plus.
3. **Déjà prêt ?** Passe directement à l'étape 5 si ces trois conditions sont réunies :
   - `git merge-base --is-ancestor <integration_branch> <branch>` réussit ;
   - le worktree est propre ;
   - l'ouvrier a rendu `DONE` avec des tests verts après son rebase.
4. **Sinon, INTEGRATION MODE.** Relance l'ouvrier du ticket par `SendMessage` s'il est joignable. Sinon, ou pour un worktree hors ticket, lance un nouvel `ork:worker` (modèle `opus`) avec :
   - `INTEGRATION MODE`, `WORKTREE: <path>` et `BRANCH: <name>` ;
   - `integration_branch`, la commande de tests et les notes ;
   - `TICKET FILE: <chemin absolu>`, s'il y a un ticket.

   Le ticket reste `in-review`. Traite le rapport selon `ork:dispatch`, puis reviens ici.
5. **Poser.** Revérifie que le dépôt principal est propre : le développeur a pu y toucher entre-temps. S'il ne l'est plus, arrête-toi comme à l'étape 1. Puis `git merge --ff-only <branch>`. Si l'avance rapide est refusée, c'est que `integration_branch` a avancé : retour à l'étape 4.
6. **Clore le ticket**, s'il y en a un : statut `done`, ligne de journal (commits, conflits), fichier déplacé dans `.ork/tickets/done/`. Puis commite-le dans le dépôt principal, sur `integration_branch`, avec les captures qu'il cite. Ne commite que lui : les autres tickets et les changements déjà indexés restent en dehors. Le pathspec couvre l'ancien chemin comme le nouveau, que le ticket ait déjà été suivi par git ou non :
   ```bash
   p=':(glob).ork/tickets/**/<id>-<slug>.md'
   git add -A -- "$p" <captures>
   git commit -m "<id>-<slug> : intégré" -- "$p" <captures>
   ```
   Suis le style de commit du dépôt s'il en a un. Si `.ork/tickets/` est ignoré par git (`git check-ignore -q .ork/tickets/done/`), ne commite rien. Puis lance les tickets `ready` dont `depends_on` le cite et qui n'attendent plus rien.
7. **Supprimer**, après avoir vérifié `git merge-base --is-ancestor <branch> <integration_branch>` :
   - `git worktree remove <path>`, puis `git branch -d <branch>` ;
   - worktree verrouillé dont le pid est mort, ou dont le pid est le tien (voir l'étape 2) : `git worktree unlock`, puis `remove` ;
   - pid vivant d'une autre session, ou session courante dans ce worktree : laisse la suppression au développeur et donne-lui les commandes.
8. **Rendre compte**, en lignes d'état, par intégration :
   - `` [OK] `<id>-<slug>` -> `<integration_branch>` ``, avec en complément les seuls conflits tranchés ; ou `` [FAIL] `<id>-<slug>` : <où ça s'est arrêté> `` ;
   - une ligne `[HINT]` par action que l'ouvrier a notée pour le développeur dans le journal ;
   - `[WARN]` si la suppression revient au développeur, avec les commandes.

   Le nombre de commits, les tests verts et la suppression du worktree vont dans le journal du ticket, pas dans la réponse.

# Reprendre un ouvrier injoignable

Si l'ouvrier n'est plus joignable (session redémarrée, identifiant perdu, `SendMessage` refusé), relance un nouvel ouvrier sur la branche existante :
1. repère son ancien worktree avec `git worktree list` ;
2. s'il contient des modifications non commitées (`git -C <path> status --porcelain`), arrête-toi et demande au développeur quoi en faire. Ne force rien ;
3. sinon, supprime l'ancien worktree avec `git worktree remove <path>`. La branche reste ;
4. lance un nouvel `ork:worker` avec `TICKET FILE: <chemin absolu>` et la ligne `EXISTING BRANCH: <name>`.

Cette procédure sert à **reprendre l'implémentation**. Pour une intégration, ne supprime rien : le INTEGRATION MODE travaille dans l'ancien worktree.

$ARGUMENTS
