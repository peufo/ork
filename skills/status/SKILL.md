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
- T-NNNN titre : résumé en une ligne, et le point délicat signalé par l'ouvrier

**En cours**
- T-NNNN titre : plan ou exécution, depuis quand

**Prêt à partir**
- T-NNNN titre : et pourquoi il attend (zones en conflit avec un ticket en cours, limite de parallélisme…)

**Idées à clarifier**
- T-NNNN titre

3. Termine par **une seule** proposition d'action, la plus utile maintenant (« Je lance T-0014 ? », « Tu veux relire T-0012 ? »).

Reste court. Si un groupe dépasse 5 tickets, montre les 5 plus prioritaires et le nombre restant.

$ARGUMENTS
