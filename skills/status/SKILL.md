---
name: status
description: Faire le point sur les tickets ouverts, au démarrage d'une session comme en cours de journée, et livrer les notifications différées.
disable-model-invocation: true
---

Si tes instructions n'ont pas de section « Un sujet à la fois », tu n'es pas le chef : réponds seulement ``[WARN] à lancer depuis le chef : `claude --agent ork:lead` `` et arrête-toi.

Le développeur veut savoir où on en est : au démarrage d'une session, ou en cours de journée. Il a choisi ce moment : c'est là que tu livres tout ce que tu as gardé en différé, c'est-à-dire les lignes `[deferred]` des journaux, dans `.ork/tickets/` comme dans `done/`. Une fois le point rendu, remplace leur marque par `[delivered]`.

1. Lis `.ork/config.md`. S'il n'existe pas, propose au développeur de lancer `/ork:init`, puis arrête-toi là.
2. Relis le frontmatter et la section `## Questions` de tous les tickets de `.ork/tickets/` (pas `done/`). Vérifie avec `git worktree list --porcelain` et `git branch --list 'ticket/*'` quelles branches de tickets existent encore. Ne te fie pas à ta mémoire de conversation. Une dépendance (`depends_on`) qui ne correspond à aucun ticket, ni dans `.ork/tickets/` ni dans `done/`, ou qui forme un cycle, va dans **À toi**.
3. Un ticket en `planning` ou `in-progress` dont l'ouvrier (`agent:`) n'apparaît pas dans `ListAgents` est **arrêté** : son ouvrier ne tourne plus. En début de session, c'est le cas de tous. Ne te fie pas à ta mémoire de ce que tu as lancé : elle a pu être compactée. Ne le relance pas d'office : signale-le.
4. Réponds en lignes d'état (voir « Les lignes d'état » dans tes instructions), une par ticket, dans l'ordre de leur tableau :
   - **arrêté** : `` [STOP] `tN-slug` : plan ou exécution, ce que dit la fin du journal ``. Le relancer, c'est donner à un nouvel ouvrier le ticket complet, journal compris, sur la branche existante (voir « Si l'ouvrier n'est plus joignable » dans tes instructions) ;
   - **intégré sans relecture** (un ticket de `done/` dont le journal note une intégration sans relecture marquée `[deferred]`) : `` [OK] `tN-slug` -> `<integration_branch>` : sans relecture `` ;
   - **en cours** : `` [RUN] `tN-slug` : plan ou exécution, depuis quand `` ;
   - **prêt à partir** : `` [DEP] `tN-slug` -> `t11-slug` `` s'il attend un ticket (`depends_on` encore ouvert, zones en conflit avec un ticket en cours), sinon `` [READY] `tN-slug` : raison `` (limite de parallélisme, `## Ask before`…) ;
   - **idée à clarifier** : `` [IDEA] `tN-slug` `` ;
   - **à savoir** (les autres lignes `[deferred]`, comme une petite découverte) : `` [NOTE] `tN-slug` : la ligne du journal `` ;
   - **worktree à surveiller** : `` [WT] `<worktree>` (`<branch>`) : avance, retard, fichiers sales, dernière activité, verrou ``, en omettant ce qui vaut zéro. Hors dépôt principal, retiens les worktrees qui ne sont liés à aucun ticket ouvert (session lancée à part, ou ticket déjà clos), qui n'ont rien à intégrer (propres, et sans avance sur `integration_branch`), ou qui portent des modifications non commitées. Pour un worktree hors ticket qui a de l'avance, ajoute `` [HINT] `intègre <name>` ``. Pour un worktree qui n'a rien à intégrer, propose de le supprimer de la même façon ;
   - **ce qui attend le développeur** (question, plan à valider, échec, ticket `in-review` avec ce qui demande sa relecture : points signalés, `remaining` de la revue auto, plan, `review: required`) : en sujets, voir l'étape 5.

```bash
git rev-list --left-right --count <integration_branch>...<branch>   # retard, avance
git -C <wt> status --short | wc -l
git -C <wt> log -1 --format=%cr
```

5. S'il y a des sujets, le premier dans l'ordre de la file devient le sujet en cours (voir « Un sujet à la fois » dans tes instructions) : les autres font chacun une ligne `[WAIT]`, et tu termines en présentant le premier selon « Présenter un sujet ». Sinon, termine par **une seule** proposition, la plus utile maintenant, présentée en sujet avec ses actions (`` `relance` : relancer `t13-supprimer-deck` ``, `` `relis` : ouvrir la relecture de `t12-bouton-enregistrer` ``).

Reste court. Au-delà de 5 lignes pour un même tag, montre les 5 plus prioritaires puis `[TAG] +N`.

$ARGUMENTS
