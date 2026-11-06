---
name: init
description: Configurer ork dans un dépôt, revoir sa configuration, ou migrer un projet de `doc/ork.md` vers `.ork/`. Détecte les commandes et la branche d'intégration, aligne les dossiers sur les conventions d'ork, vérifie que CLAUDE.md donne aux ouvriers ce dont ils ont besoin, puis écrit `.ork/config.md` après accord.
disable-model-invocation: true
---

Le développeur installe ork dans ce dépôt, ou veut revoir sa configuration. C'est un moment de mise en place : tu peux poser plusieurs questions, mais tu n'écris **rien** avant son accord.

$ARGUMENTS

# Les conventions d'ork

| Chemin | Rôle |
|---|---|
| `.ork/config.md` | configuration : frontmatter (`docs_dir`, `test_command`, `integration_branch`, `max_parallel`, `auto_review`, `review`, `confidence_threshold`), puis les sections `## Plan required when`, `## Review required when`, `## Ask before`, `## No need to ask` et `## Notes` |
| `.ork/tickets/` | les tickets ouverts |
| `.ork/tickets/captures/` | les captures d'écran jointes |
| `.ork/tickets/done/` | les tickets terminés |
| `<docs_dir>/story/` | les stories, une par fichier |

`.ork/` appartient à ork. La documentation du projet, stories comprises, reste dans son propre dossier : `docs_dir`, `doc/` par défaut. La micro-app n'affiche que ces deux dossiers.

Ce qui est propre au projet ne va pas dans `.ork/config.md`, mais dans le `CLAUDE.md` du dépôt, que tous les ouvriers lisent : préparation d'un worktree, fichiers générés, vérifications supplémentaires, style de commit. `.ork/config.md` ne garde que la configuration d'ork et les écarts aux conventions.

# 1. Point de départ

- Si `.ork/config.md` existe, tu es en **révision** : lis-le, et ne propose ensuite que les écarts entre ce fichier et ce que tu détectes.
- Sinon, si `doc/ork.md` existe, c'est une **migration** : le projet suit l'ancien emplacement. Lis ce fichier. Tu proposes la migration, puis tu continues comme une révision.
- Sinon, c'est une **installation**.
- Note la branche courante et `git status --short`. Ne touche pas aux modifications que tu y trouves.
- Lis `.claude/settings.json` et `.claude/settings.local.json` s'ils existent. Tu y cherches trois réglages :
  - `"agent": "ork:lead"` : le chef devient la session par défaut du projet ;
  - `"permissions": { "deny": ["Agent(fork)"] }` : un filet de sécurité contre les forks, qui héritent de toute la conversation du chef ;
  - `"statusLine": { "type": "command", "command": "sh ${CLAUDE_PLUGIN_ROOT}/scripts/statusline.sh" }` : ce qui attend le développeur dans les tickets, toujours visible dans le terminal. Elle demande `jq`. Si une autre `statusLine` existe, ne la remplace pas : signale-la. Regarde aussi `~/.claude/settings.json`, où elle a pu être posée pour tous les projets.

# 2. Enquête

Lance **un seul** `ork:scout`, avec ces questions. Garde sa conclusion, pas le détail.

1. **Commandes :** quelles commandes vérifient le projet (types, lint, tests, contrat ou code généré) ? Comment les lance la CI (`.github/workflows/`, etc.) ? Propose une `test_command` unique, enchaînée avec `&&`.
2. **Branche d'intégration :** sur quelle branche le travail quotidien est-il posé ? Indices : branche courante, branche par défaut du remote, branches de la CI, conventions écrites dans `CLAUDE.md` ou dans la doc.
3. **Dossiers :** où vit la documentation du projet (`doc/`, `docs/`, `documentation/`…) ? C'est la valeur de `docs_dir`. Quels dossiers existants jouent l'un des rôles du tableau ci-dessus sous un autre nom (par exemple `docs/stories/`, `tickets/`) ? Pour chacun, liste **toutes** ses références dans les fichiers suivis, sauf `.ork/tickets/done/` (`git grep -n -I '<path>'`), en distinguant :
   - les chemins écrits en clair, qu'on peut remplacer mécaniquement ;
   - les chemins construits ou indirects (concaténation, glob, variable).
4. **`CLAUDE.md` :** décrit-il ces quatre points ? Pour chaque point absent, ce que tu as pu déduire du dépôt :
   - la préparation d'un worktree (dépendances, fichiers d'environnement non versionnés) ;
   - les fichiers générés à régénérer plutôt que fusionner (fichiers de verrouillage, contrats, clients) ;
   - les vérifications supplémentaires selon les zones touchées ;
   - le style de commit (à partir de `git log --format=%s -30`).
5. **Doublons :** des skills, commandes ou agents du projet (`.claude/skills/`, `.claude/commands/`, `.claude/agents/`) recoupent-ils ork (intégration de branches, tickets, suivi d'état, worktrees) ? Pour chacun, ce qu'il contient de propre au projet.
6. **Ressources partagées :** une ressource commune rend-elle dangereux deux ouvriers en parallèle ? Exemples : une base de test unique, un port fixe, un dossier de sortie partagé.
7. **Tickets dans git :** s'il y a déjà des tickets, sont-ils versionnés, ou ignorés par `.gitignore` ?
8. **Zones sensibles :** quels dossiers portent l'authentification, les droits, les paiements, les migrations ou la suppression de données ?
9. **Migration seulement :** liste toutes les références à l'ancien emplacement, tickets de `done/` compris :
   - les chemins depuis la racine : `git grep -n -I --untracked -E 'doc/tickets|doc/ork\.md'`, plus `grep -rn` dans `doc/tickets/` si les tickets sont ignorés par git ;
   - les liens relatifs qui traversent la frontière : depuis un ticket vers le reste de `doc/` (`](../story/a.md)`), depuis un document vers un ticket (`](../tickets/t1.md)`) ;
   - les références construites ou indirectes (glob, filtre, variable, script).

# 3. Proposition

Présente tout en **un seul message**, en groupant :

- **La migration**, si le projet suit l'ancien emplacement : ce qui bouge (`doc/ork.md` → `.ork/config.md`, `doc/tickets/` → `.ork/tickets/`), ce qui reste (les stories et le reste de la documentation), le nombre de références à mettre à jour, archives comprises.
- **La configuration proposée :** le frontmatter complet, puis les règles « Plan required when » adaptées au dépôt (modèle de données, API publique, auth, build, nombre de fichiers). Pour `max_parallel`, propose `1` la première semaine. Si une ressource partagée a été trouvée, dis pourquoi le parallélisme est risqué. Pour la revue, propose `auto_review: [/simplify]` et `review: always` la première semaine : le développeur juge la qualité de la revue auto avant de passer à `per-ticket`. Pour la même raison, laisse `confidence_threshold` absent la première semaine : le développeur voit tous les points avec leur note, et juge à partir de quelle note il peut les laisser passer. Propose des règles `## Review required when` à partir des zones sensibles trouvées. Laisse `## Ask before` et `## No need to ask` vides : elles se remplissent avec les corrections du développeur.
- **Les écarts de dossiers :** chemin actuel → convention, et le nombre de références à mettre à jour.
- **Ce qui manque à `CLAUDE.md` :** le texte que tu proposes d'y ajouter, dans une section du type « Git, worktrees et intégration ».
- **Les doublons :** ce qu'il faut en garder (dans `CLAUDE.md`) et ce que tu proposes de supprimer.
- **Les réglages recommandés** qui manquent, s'il y en a.

Puis fais trancher avec `AskUserQuestion`, en quatre questions au plus par appel :

- **La migration**, si le projet suit l'ancien emplacement : « Migrer vers `.ork/` (Recommandé) », ou « Pas maintenant ». Dans ce second cas, n'applique rien et arrête-toi : le chef, la micro-app et la status line ne lisent plus que `.ork/`.
- **Un dossier hors convention, par question :**
  - « Renommer et mettre à jour les références (Recommandé) », toujours en premier ;
  - « Garder le dossier, ajouter une note dans `.ork/config.md` ».

  Si des références sont construites ou indirectes, dis-le dans la description : elles ne se remplacent pas mécaniquement, et un ouvrier devra s'en charger dans un ticket.
- **Suivi git des tickets :** « Commités (Recommandé) », parce que l'historique des tickets suit le code et survit à un changement de machine ; ou « Ignorés par git ».
- **Les ajouts à `CLAUDE.md` et le sort des doublons**, si tu en as trouvé.
- **Les réglages recommandés**, s'il en manque : où les écrire ?
  - « `.claude/settings.local.json` (Recommandé) » : personnel, non commité ;
  - « `.claude/settings.json` » : commité, donc imposé à tous ceux qui ouvrent le dépôt.
- **La configuration**, si un choix reste ambigu (par exemple deux branches d'intégration plausibles).

# 4. Appliquer

Dans cet ordre, uniquement ce qui a été accepté.

0. **Migrer vers `.ork/` :**
   - si un ticket est en `planning` ou `in-progress` et que son ouvrier (`agent:`) apparaît dans `ListAgents`, arrête-toi : la migration attend la fin de son travail ;
   - déplace, avec `git mv` si le chemin est suivi, `mv` sinon :
     ```bash
     mkdir -p .ork
     git mv doc/tickets .ork/tickets
     git mv doc/ork.md .ork/config.md
     ```
     Les tickets non commités suivent le dossier, avec leurs modifications ;
   - remplace chaque référence que l'enquête a trouvée, y compris dans `.gitignore` et dans les tickets de `done/`. C'est la seule exception à la règle des archives, parce que leurs liens casseraient :
     - `doc/tickets/` → `.ork/tickets/` et `doc/ork.md` → `.ork/config.md` ;
     - un lien relatif qui traverse la frontière devient un chemin depuis la racine : `../story/a.md` → `doc/story/a.md`, `../tickets/t1.md` → `.ork/tickets/t1.md` ;
   - **vérifie** avec `git grep -n -I --untracked -E 'doc/tickets|doc/ork\.md'`. Le résultat doit être vide. Traite une référence construite comme à l'étape 1 ;
   - ajoute `docs_dir` au frontmatter de `.ork/config.md`. Ne déplace rien d'autre dans `doc/`.
1. **Renommer un dossier :**
   - `git mv <old> <new>` si le dossier est suivi, `mv` sinon ;
   - remplace chaque chemin écrit en clair que l'enquête a trouvé. Ce sont des remplacements de texte exacts : ne modifie rien d'autre dans ces fichiers ;
   - **vérifie** avec `git grep -n -I --untracked '<old>' -- ':!.ork/tickets/done'`. Le résultat doit être vide. Sinon, corrige si c'est un chemin en clair oublié ; si c'est une référence construite, laisse-la et crée un ticket `ready`, qui la liste, pour qu'un ouvrier s'en occupe ;
   - ne réécris jamais l'historique git, ni les tickets de `done/` : ce sont des archives.
2. **Garder un dossier :** ajoute dans `## Notes` de `.ork/config.md` une ligne comme « **Stories :** elles sont dans `docs/stories/`, pas dans `docs/story/` ».
3. **Écrire `.ork/config.md`.** En révision, ne change que ce qui a été accepté et garde le reste tel quel.
4. **Créer les dossiers** `.ork/tickets/`, `.ork/tickets/captures/` et `.ork/tickets/done/`. S'ils sont commités, ajoute un `.gitkeep` dans chaque dossier vide. S'ils sont ignorés, ajoute `.ork/tickets/` au `.gitignore`. Ne crée pas `docs_dir` : le chef le créera avec la première story.
5. **Compléter `CLAUDE.md`** avec le texte accepté.
6. **Ajouter les réglages** acceptés dans le fichier choisi. Fusionne-les avec le contenu existant : ajoute à `permissions.deny` sans remplacer la liste, et garde les autres clés. Vérifie ensuite que le fichier reste du JSON valide.
7. **Traiter les doublons :** une fois leur contenu propre au projet passé dans `CLAUDE.md`, supprime ceux dont la suppression a été acceptée.

# 5. Commiter

Propose un commit de ces seuls changements, sur la branche courante, dans le style du dépôt. Laisse en dehors toute modification qui existait avant l'init, sauf en migration : les tickets et la configuration déplacés partent entiers, avec leurs modifications non commitées. Dis-le au développeur avant de commiter. Ne pousse jamais. Si le développeur préfère commiter lui-même, donne-lui la liste des fichiers.

# 6. Rendre compte

En six lignes au plus :
- la migration, si elle a eu lieu, et le nombre de références mises à jour ;
- la configuration retenue ;
- les dossiers renommés et le nombre de références mises à jour (ou la note ajoutée) ;
- ce qui a été ajouté à `CLAUDE.md` et ce qui a été supprimé, ainsi que les réglages ajoutés ;
- les tickets créés, s'il y en a ;
- la suite : si tu as ajouté `agent`, redémarrer Claude Code pour qu'il s'applique ; `/reload-plugins` si tu as modifié des skills du projet ; si `docs_dir` a changé alors que la micro-app tourne, la relancer, car elle ne lit cette clé qu'au démarrage ; puis `/ork:ticket` ou une story pour commencer.
