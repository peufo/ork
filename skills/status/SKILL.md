---
name: status
description: Faire le point sur les tickets ouverts, au démarrage d'une session comme en cours de journée.
---

Si tes instructions n'ont pas de section « Un sujet à la fois », tu n'es pas le chef : réponds seulement ``[WARN] à lancer depuis le chef, tape /ork`` et arrête-toi.

Le développeur veut savoir où on en est : au démarrage d'une session, ou en cours de journée.

1. Lis `.ork/config.md`. S'il n'existe pas, propose au développeur de lancer `/ork:init`, puis arrête-toi là. Si `.ork/tickets/.stop` existe et qu'aucun ouvrier ne tourne, l'arrêt précédent n'a pas été terminé : supprime ce fichier et ajoute une ligne `[NOTE]`.
2. Relis le frontmatter et la section `## Questions` de tous les tickets de `.ork/tickets/` (pas `done/`). Vérifie avec `git worktree list --porcelain` et `git branch --list 'ticket/*'` quelles branches de tickets existent encore, et avec `git branch --show-current` si le dépôt principal est sur une branche `review/*`. Ne te fie pas à ta mémoire de conversation. Repère une dépendance (`depends_on`) qui ne correspond à aucun ticket, ni dans `.ork/tickets/` ni dans `done/`, ou qui forme un cycle.
3. Un ticket en `planning` ou `in-progress` dont l'ouvrier (`agent:`) n'apparaît pas dans `ListAgents` est **arrêté** : son ouvrier ne tourne plus. Ne le relance pas d'office : signale-le.
4. Le panneau d'ork montre déjà les tickets en cours, en attente et prêts : ne les répète pas. Réponds en lignes d'état, seulement pour ce qu'il ne montre pas :
   - **relecture ouverte** : `` [NOTE] `tN-slug` : relecture ouverte sur `review/tN` ``. Elle est le premier sujet : reprends-la selon `ork:review` ;
   - **arrêté** : `` [STOP] `tN-slug` : plan ou exécution, ce que dit la fin du journal ``. Le relancer se fait selon `ork:integrate` ;
   - **dépendance cassée** : `` [WARN] `tN-slug` : <la dépendance> `` ;
   - **worktree à surveiller** : `` [WT] `<worktree>` (`<branch>`) : avance, retard, fichiers sales, dernière activité, verrou ``, en omettant ce qui vaut zéro. Hors dépôt principal, retiens les worktrees qui ne sont liés à aucun ticket ouvert, qui n'ont rien à intégrer (propres, et sans avance sur `integration_branch`), ou qui portent des modifications non commitées. Pour un worktree hors ticket qui a de l'avance, ajoute `` [HINT] `intègre <name>` ``. Pour un worktree qui n'a rien à intégrer, propose de le supprimer de la même façon ;

```bash
git rev-list --left-right --count <integration_branch>...<branch>   # retard, avance
git -C <wt> status --short | wc -l
git -C <wt> log -1 --format=%cr
```

5. S'il y a des sujets (question, plan à valider, échec, relecture), présente le premier dans l'ordre de la file, selon « Présenter un sujet ». Sinon, termine par **une seule** proposition, la plus utile maintenant, présentée en sujet avec ses actions.

Reste court.

$ARGUMENTS
