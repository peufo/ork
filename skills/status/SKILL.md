---
name: status
description: Faire le point sur tous les tickets ouverts et livrer les notifications différées.
disable-model-invocation: true
---

Le développeur veut savoir où on en est. Il a choisi ce moment : c'est là que tu livres tout ce que tu as gardé en différé.

1. Relis le frontmatter de tous les tickets de `doc/tickets/` (pas `done/`). Ne te fie pas à ta mémoire de conversation.
2. Réponds avec cette structure, en omettant les groupes vides :

**À toi** (ce qui attend une décision ou une réponse)
- T-NNNN titre : la question ou le plan à valider, en une ligne, avec ta recommandation

**À relire**
- T-NNNN titre : résumé en une ligne, et ce qui demande ta relecture (points signalés, `remaining` de la revue auto, plan, `review: required`)

**Intégré sans relecture** (depuis le dernier point)
- T-NNNN titre : résumé en une ligne

**En cours**
- T-NNNN titre : plan ou exécution, depuis quand

**Prêt à partir**
- T-NNNN titre : et pourquoi il attend (zones en conflit avec un ticket en cours, limite de parallélisme…)

**Idées à clarifier**
- T-NNNN titre

**Worktrees à surveiller** (ce groupe n'apparaît que si l'un d'eux est concerné)
`git worktree list --porcelain`, hors dépôt principal. Retiens ceux qui :
- ne sont liés à aucun ticket ouvert (session lancée à part, ou ticket déjà clos) ;
- n'ont rien à intégrer (propres, et sans avance sur `integration_branch`) ;
- portent des modifications non commitées.

| worktree | branche | avance / retard | fichiers sales | dernière activité | verrou |
|---|---|---|---|---|---|

```bash
git rev-list --left-right --count <integration_branch>...<branch>   # retard, avance
git -C <wt> status --short | wc -l
git -C <wt> log -1 --format=%cr
```

Pour un worktree hors ticket qui a de l'avance, propose « intègre <name> ». Pour un worktree qui n'a rien à intégrer, propose de le supprimer.

3. Si le groupe « À toi » n'est pas vide, son premier élément devient le sujet en cours (voir « Un sujet à la fois » dans tes instructions) : termine en le présentant. Sinon, termine par **une seule** proposition d'action, la plus utile maintenant (« Je lance T-0014 ? », « Tu veux relire T-0012 ? »).

Reste court. Si un groupe dépasse 5 tickets, montre les 5 plus prioritaires et le nombre restant.

$ARGUMENTS
