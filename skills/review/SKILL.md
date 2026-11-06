---
name: review
description: Mener la relecture guidée d'un ticket ork en in-review, sur une branche dédiée du dépôt principal. À charger par le chef quand le développeur demande à relire un ticket.
---

Un ticket `in-review` attend le développeur. Quand il demande à le relire, il devient le sujet en cours.

La relecture se fait sur une branche dédiée du dépôt principal, parce que le panneau de modifications de l'app suit la branche de ce dossier. `integration_branch` ne bouge qu'après la validation. Une seule relecture à la fois : tant qu'elle est ouverte, rien ne se pose sur `integration_branch`. Les ouvriers continuent dans leurs worktrees, et tu peux lancer des tickets.

# Ouvrir la relecture

Regarde `git branch --show-current` dans le dépôt principal :
- `review/tN` de ce ticket : la relecture est déjà ouverte (compaction, nouvelle session). Fais `git merge --ff-only @{u}`, puis reprends au premier point non coché ;
- `review/tM` d'un autre ticket : une autre relecture est ouverte. Présente-la : celle-ci attend sa fermeture ;
- `integration_branch` : continue ci-dessous ;
- autre chose : avertis en nommant la branche, et arrête-toi.

1. **Dépôt propre**, avec les mêmes règles que l'étape 1 de `ork:integrate`. Sinon, avertis et arrête-toi.
2. **Branche à jour.** Si `git merge-base --is-ancestor <integration_branch> <branch>` échoue, fais-la rebaser selon l'étape 4 de `ork:integrate`, et reprends ici à son rapport. Le développeur relit ainsi ce qui sera posé.
3. **Basculer :** `git switch -c review/tN --track <branch>`. Les tickets modifiés suivent. Si git refuse, avertis et arrête-toi, sans contourner.
4. **Montrer le code :** ouvre le panneau avec `show_pane` (`pane: diff`, sans `diff_scope`), une seule fois.

Ne lance ni migration ni commande du projet dans le dépôt principal. Une action notée pour le développeur dans le journal du ticket va dans une ligne `[HINT]` de la vue d'ensemble : il décide s'il la fait maintenant.

# Mener la relecture

Chaque message suit « Présenter un sujet », avec l'étape de l'en-tête indiquée entre parenthèses.

1. **Vue d'ensemble** (étape `N à relire sur M`) :
   - une ligne : ce que fait le ticket ;
   - une ligne de volume : `6 fichiers · +182 −40 · 3 commits`, tirée de `git diff --shortstat <integration_branch>...<branch>` et `git rev-list --count <integration_branch>..<branch>` ;
   - si `integration_branch` n'est pas la branche par défaut du dépôt, une ligne pour le panneau, qui compare d'abord à celle-ci : `Dans le panneau : Comparer à <integration_branch>` ;
   - une puce `Décision :` par choix notable de l'ouvrier, seulement s'il y en a ;
   - la liste numérotée de **tous** les points de `## Review`, triés par note croissante, chacun en `` [N/10] `path:line` : phrase courte ``. Un point coché d'office porte `✓` après sa note. Un point sans note s'écrit `[?/10]`, vient en tête et n'est jamais coché d'office.

   ```markdown
   ### 👀 [t27-colonnes](.ork/tickets/t27-colonnes.md) · 2 à relire sur 4
   Remplace le JSON `state_before` de `review_logs` par 9 colonnes typées.
   6 fichiers · +182 −40 · 3 commits
   Dans le panneau : Comparer à dev

   1. [5/10] `migration.sql:17` : casts vers `TIMESTAMP(3)`
   2. [6/10] `decks.md:25` : la story cite encore `ReviewLog.stateBefore`
   3. [8/10] ✓ `schema.ts:40` : noms des colonnes, validés au plan

   `relis` : commencer par le point 1
   `3` : voir un point précis
   `attends` : fermer la relecture sans intégrer
   ```
2. **Un point par message** (étape `point 1/2 · 5/10`), dans l'ordre de la liste, en sautant les points `✓`. Un numéro tapé par le développeur ouvre ce point, coché ou non. Écris le fichier et sa ligne, suivis de la phrase de l'ouvrier, puis l'action `ok`, sans rien après. N'ouvre aucun panneau à chaque point. Si le développeur demande un fichier, ouvre-le avec `show_pane` (`pane: file`, chemin dans le dépôt principal, `line`). Ne relis pas le code toi-même.

   ```markdown
   ### 👀 [t27-colonnes](.ork/tickets/t27-colonnes.md) · point 1/2 · 5/10
   `migration.sql:17` : casts vers `TIMESTAMP(3)`

   `ok` : valider et passer au point 2
   ```
3. **Sa réponse** : « ok » coche le point dans `## Review` et passe au suivant. Après une compaction, la relecture reprend au premier point non coché. Une remarque va dans `## Review`, puis à l'ouvrier par `SendMessage` (« corrige … »). À son rapport, fais `git merge --ff-only @{u}` dans le dépôt principal, puis reprends sur les seuls points nouveaux ou touchés. Si l'avance rapide est refusée, l'ouvrier a réécrit ses commits : avertis, et arrête-toi.
4. **Après le dernier point** (étape `relecture terminée`) : les actions `intègre` et `attends`.

# Fermer la relecture

Ferme-la sur `intègre`, sur `attends`, et lors d'un arrêt propre :

1. `git switch <integration_branch>` ;
2. `git branch -d review/tN`. Cette suppression sans force réussit, car la branche suit celle du ticket. Si git refuse, un commit n'existe que sur `review/tN` : avertis en le nommant, et laisse-la ;
3. `intègre` : suis `ork:integrate`. `attends` : le ticket reste `in-review`, et les cases déjà cochées restent ;
4. reprends les poses notées `pose en attente` dans le journal des tickets `in-review`, selon `ork:integrate`.

$ARGUMENTS
