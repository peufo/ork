---
name: ui
description: Ouvrir la micro-app ork, qui affiche les tickets en kanban et les documents du projet en lecture seule.
disable-model-invocation: true
---

Lance la micro-app pour le dépôt courant :

```bash
sh "${CLAUDE_PLUGIN_ROOT}/scripts/ork-app.sh" --detach --open
```

Elle tourne en arrière-plan et survit à cette session. Un second lancement réutilise le serveur déjà ouvert.

Réponds par une seule ligne : `` [HINT] <url> ``, avec l'URL affichée par la commande. Si la commande échoue, rends son message d'erreur tel quel avec une ligne `[FAIL]`.

Pour l'arrêter, si le développeur le demande : `sh "${CLAUDE_PLUGIN_ROOT}/scripts/ork-app.sh" --stop`.

$ARGUMENTS
