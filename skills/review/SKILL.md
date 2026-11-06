---
name: review
description: Mener la relecture guidée d'un ticket ork en in-review. À charger par le chef quand le développeur demande à relire un ticket.
---

Un ticket `in-review` attend le développeur. Quand il demande à le relire, il devient le sujet en cours. Chaque message suit « Présenter un sujet », avec l'étape de l'en-tête indiquée entre parenthèses.

1. **Vue d'ensemble** (étape `N à relire sur M`) :
   - une ligne : ce que fait le ticket ;
   - une ligne de volume : `6 fichiers · +182 −40 · 3 commits`, tirée de `git diff --shortstat <integration_branch>...<branch>` et `git rev-list --count <integration_branch>..<branch>` ;
   - une puce `Décision :` par choix notable de l'ouvrier, seulement s'il y en a ;
   - la liste numérotée de **tous** les points de `## Review`, triés par note croissante, chacun en `` [N/10] `path:line` : phrase courte ``. Un point coché d'office porte `✓` après sa note. Un point sans note s'écrit `[?/10]`, vient en tête et n'est jamais coché d'office.

   ```markdown
   ### 👀 [t27-colonnes](.ork/tickets/t27-colonnes.md) · 2 à relire sur 4
   Remplace le JSON `state_before` de `review_logs` par 9 colonnes typées.
   6 fichiers · +182 −40 · 3 commits

   1. [5/10] `migration.sql:17` : casts vers `TIMESTAMP(3)`
   2. [6/10] `decks.md:25` : la story cite encore `ReviewLog.stateBefore`
   3. [8/10] ✓ `schema.ts:40` : noms des colonnes, validés au plan

   `relis` : commencer par le point 1
   `3` : voir un point précis
   `sujet suivant` : passer à `t28-traduire`
   ```
2. **Un point par message** (étape `point 1/2 · 5/10`), dans l'ordre de la liste, en sautant les points `✓`. Un numéro tapé par le développeur ouvre ce point, coché ou non. Écris le fichier et sa ligne, suivis de la phrase de l'ouvrier.
3. **Sa réponse** : « ok » coche le point dans `## Review` et passe au suivant. Après une compaction, la relecture reprend au premier point non coché. Une remarque va dans `## Review`, puis à l'ouvrier par `SendMessage` (« corrige … »). La relecture reprend à son rapport, sur les seuls points nouveaux ou touchés.
4. **Après le dernier point** (étape `relecture terminée`) : les actions `intègre` et `attends`.

**Montrer le code.** Le développeur le lit dans l'app. Avec l'outil `show_pane`, ouvre le panneau `diff` sur le commit de l'ouvrier (`diff_scope` : son SHA), ou le panneau `file` à la ligne d'un point (`path` dans le worktree, `line`). Ne relis pas le code toi-même. Un commentaire du développeur dans le diff te parvient comme une remarque : note-le dans `## Review` et transmets-le à l'ouvrier.

$ARGUMENTS
