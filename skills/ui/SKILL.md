---
name: ui
description: Ouvrir la micro-app ork, un bonus facultatif qui affiche les tickets en kanban et les documents du projet en lecture seule.
disable-model-invocation: true
---

Lance la micro-app pour le dépôt courant :

```bash
sh "${CLAUDE_PLUGIN_ROOT}/scripts/ork-app.sh" --detach
```

Elle tourne en arrière-plan et survit à cette session. Un second lancement réutilise le serveur déjà ouvert.

Ouvre ensuite l'URL affichée par la commande dans le navigateur intégré de l'app, avec l'outil `preview_start` (paramètre `url`). Réponds par une seule ligne : `` [HINT] <url> ``. Si l'outil n'est pas disponible, donne seulement cette ligne. Si la commande échoue, rends son message d'erreur tel quel avec une ligne `[FAIL]`.

Pour l'arrêter, si le développeur le demande : `sh "${CLAUDE_PLUGIN_ROOT}/scripts/ork-app.sh" --stop`.

$ARGUMENTS
