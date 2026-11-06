---
name: stop
description: Terminer la session d'orchestration au prochain état stable, pour que le développeur puisse fermer sans rien perdre. À charger par le chef quand le développeur tape /ork:stop, demande de s'arrêter, de finir proprement ou de fermer bientôt.
---

Le développeur veut pouvoir fermer sans rien perdre. Tu ne lances plus rien, tu laisses ce qui tourne se poser, puis tu lui dis quand il peut fermer.

Si tes instructions n'ont pas de section « Un sujet à la fois », tu n'es pas le chef : réponds seulement ``[WARN] à lancer depuis le chef, tape /ork`` et arrête-toi.

# État stable

L'état est stable quand tout cela est vrai :
- aucun ouvrier ne travaille (`ListAgents`) ;
- aucun rebase n'est en cours dans un worktree (`git -C <path> status`) ;
- chaque ticket reflète la réalité : statut, journal, branche ;
- aucune relecture n'est ouverte : le dépôt principal est sur `integration_branch` ;
- le dépôt principal n'a pas changé à cause de toi. Ce qui y traîne et qui n'est pas à toi se signale, sans y toucher.

Le travail non intégré n'empêche pas l'état d'être stable : il attend sur sa branche.

# Procédure

1. **Note la demande.** Écris `.ork/tickets/.stop` avec l'outil Write, contenant la date. Il survit à une compaction, et il signale l'arrêt dans le panneau.
2. **Ne lance plus rien** tant que ce fichier existe : ni ticket `ready`, ni dépendance débloquée, ni intégration nouvelle. Une intégration dont l'ouvrier travaille déjà va à son terme, sans que tu la poses si elle demande encore une décision.
3. **Préviens chaque ouvrier actif** par `SendMessage` : `STOP AT NEXT STABLE POINT`. Il finit le geste en cours, commite sur sa branche et rend son rapport.
4. **Réponds par une phrase :** qui tu attends, puis arrête ton tour. Les rapports te réveilleront.
5. **À chaque rapport**, traite-le selon `ork:dispatch` avec ces différences :
   - `DONE` : pas d'intégration, même si les conditions d'intégration directe sont réunies ;
   - `PAUSED` : le ticket garde son statut ;
   - `BLOCKED` ou `PLAN READY` : ne pose pas la question maintenant ;
   - fin sans rapport : note-le dans le journal.
6. **Si une relecture est ouverte**, ferme-la selon `ork:review`, une fois que plus aucun ouvrier ne la corrige : le dépôt principal revient sur `integration_branch`, et les cases cochées restent.
7. **Quand l'état est stable**, supprime `.ork/tickets/.stop`, puis rends le bilan.

Si le développeur change d'avis (« reprends », « continue »), supprime `.ork/tickets/.stop` et reprends normalement.

Ne force jamais un arrêt. Si un ouvrier ne rend pas la main après un long moment, dis-le et laisse le développeur décider : l'attendre, ou l'arrêter lui-même depuis le panneau des tâches de l'app.

# Bilan

En lignes d'état, puis une phrase qui autorise à fermer :

```markdown
[OK] état stable, tu peux fermer
[NOTE] `t33-velocites` : en pause, reprise possible sur sa branche
[NOTE] `t14-export-pdf` : question en attente, pour la prochaine session
[WARN] dépôt principal : `doc/notes.md` modifié, pas par moi

À la reprise, `/ork` repartira de là.
```

Ce bilan termine le mode chef : pour la suite de cette conversation, tu n'es plus le chef.

N'écris pas ce que le panneau montre déjà. Ne cite que ce qu'il faudra reprendre, et ce qui pourrait surprendre.

$ARGUMENTS
