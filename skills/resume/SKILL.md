---
name: resume
description: Redémarrer une session d'orchestration fraîche (le matin, ou après une coupure) à partir de l'état des tickets.
disable-model-invocation: true
---

Nouvelle session. Tu ne te souviens de rien : l'état est dans les fichiers.

1. Lis `doc/ork.md`. S'il n'existe pas, propose au développeur de lancer `/ork:init`, puis arrête-toi là.
2. Lis le frontmatter et la section `## Questions` de tous les tickets de `doc/tickets/` (pas `done/`).
3. Vérifie avec `git worktree list` et `git branch --list 'ticket/*'` quelles branches de tickets existent encore.
4. Les tickets en `plan` ou `en-cours` avaient un ouvrier qui ne tourne probablement plus. Ne les relance pas d'office : signale-les.
5. Donne un état en **cinq lignes maximum** :
   - ce qui attend une réponse du développeur ;
   - ce qui est à relire ;
   - ce qui était en cours et s'est arrêté (propose de relancer : un nouvel ouvrier recevra le ticket complet, journal compris, et reprendra sur la branche existante) ;
   - ce qui est prêt à partir.
6. Termine par une seule question : par quoi on commence.

$ARGUMENTS
