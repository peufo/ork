---
name: ticket
description: Capturer une remarque ou une capture d'écran et la transformer en ticket, sans interrompre le travail en cours.
disable-model-invocation: true
---

Le développeur te signale quelque chose en passant. Il est probablement concentré sur autre chose : traite-le vite et sans le retenir.

Remarque :

$ARGUMENTS

(Une capture d'écran peut aussi être jointe au message.)

1. Si une capture d'écran est jointe et qu'elle n'existe pas encore sur disque, enregistre-la dans `doc/tickets/captures/` sous un nom daté, pour que les ouvriers puissent la lire.
2. Vérifie qu'aucun ticket ouvert dans `doc/tickets/` ne couvre déjà ce sujet. Si c'est le cas, ajoute la remarque à son contexte et à son journal au lieu de créer un doublon.
3. Si tu ne sais pas quelles zones du code sont concernées, lance `ork:scout` avec la remarque et le chemin de la capture. Ne garde que sa conclusion.
4. Crée le ticket au format habituel, en `pret` si les critères de fin sont clairs, sinon en `idee`.
5. Réponds en **une ou deux lignes** : numéro, titre, statut, et la seule question éventuelle.

Ne lance pas le ticket depuis cette commande, sauf s'il est `pret` et que les règles de parallélisme le permettent. Dans ce cas, lance-le et dis-le dans la même ligne.
