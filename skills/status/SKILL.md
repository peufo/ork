---
name: status
description: Faire le point sur les tickets ouverts, au démarrage d'une session comme en cours de journée, et livrer les notifications différées.
disable-model-invocation: true
---

Le développeur veut savoir où on en est : au démarrage d'une session, ou en cours de journée. Il a choisi ce moment : c'est là que tu livres tout ce que tu as gardé en différé, c'est-à-dire les lignes `[deferred]` des journaux, dans `doc/tickets/` comme dans `done/`. Une fois le point rendu, remplace leur marque par `[delivered]`.

1. Lis `doc/ork.md`. S'il n'existe pas, propose au développeur de lancer `/ork:init`, puis arrête-toi là.
2. Relis le frontmatter et la section `## Questions` de tous les tickets de `doc/tickets/` (pas `done/`). Vérifie avec `git worktree list --porcelain` et `git branch --list 'ticket/*'` quelles branches de tickets existent encore. Ne te fie pas à ta mémoire de conversation. Une dépendance (`depends_on`) qui ne correspond à aucun ticket, ni dans `doc/tickets/` ni dans `done/`, ou qui forme un cycle, va dans **À toi**.
3. Un ticket en `planning` ou `in-progress` dont l'ouvrier (`agent:`) n'apparaît pas dans `ListAgents` est **arrêté** : son ouvrier ne tourne plus. En début de session, c'est le cas de tous. Ne te fie pas à ta mémoire de ce que tu as lancé : elle a pu être compactée. Ne le relance pas d'office : signale-le.
4. Réponds avec cette structure, en omettant les groupes vides :

**À toi** (ce qui attend une décision ou une réponse)
- `tN-slug` : la question ou le plan à valider, en une ligne, avec ta recommandation

**Arrêtés**
- `tN-slug` : plan ou exécution, ce que dit la fin du journal. Propose de relancer : un nouvel ouvrier recevra le ticket complet, journal compris, et reprendra sur la branche existante (voir « Si l'ouvrier n'est plus joignable » dans tes instructions)

**À relire**
- `tN-slug` : résumé en une ligne, et ce qui demande ta relecture (points signalés, `remaining` de la revue auto, plan, `review: required`)

**Intégré sans relecture** (les tickets de `done/` dont le journal note une intégration sans relecture marquée `[deferred]`)
- `tN-slug` : résumé en une ligne

**À savoir** (les autres lignes `[deferred]`, comme une petite découverte)
- `tN-slug` : la ligne du journal

**En cours**
- `tN-slug` : plan ou exécution, depuis quand

**Prêt à partir**
- `tN-slug` : et pourquoi il attend (dépend de `t11-slug` encore `in-progress`, zones en conflit avec un ticket en cours, limite de parallélisme…)

**Idées à clarifier**
- `tN-slug`

**Worktrees à surveiller** (ce groupe n'apparaît que si l'un d'eux est concerné)
Hors dépôt principal, retiens les worktrees qui :
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

5. Si le groupe « À toi » n'est pas vide, son premier élément devient le sujet en cours (voir « Un sujet à la fois » dans tes instructions) : termine en le présentant selon « Présenter un sujet ». Sinon, termine par **une seule** proposition d'action, la plus utile maintenant (« Je relance `t13-supprimer-deck` ? », « Tu veux relire `t12-bouton-enregistrer` ? »).

Reste court. Si un groupe dépasse 5 tickets, montre les 5 plus prioritaires et le nombre restant.

$ARGUMENTS
