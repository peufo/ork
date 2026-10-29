---
name: init
description: Onboarding d'ork dans un dépôt, ou révision de sa configuration. Détecte les commandes et la branche d'intégration, aligne les dossiers sur les conventions d'ork, vérifie que CLAUDE.md donne aux ouvriers ce dont ils ont besoin, puis écrit `doc/ork.md` après accord. À charger uniquement dans une session d'orchestration ork (`ork:lead`), dans deux cas : `doc/ork.md` n'existe pas et le développeur accepte l'onboarding, ou il demande de revoir la configuration d'ork. Jamais dans une session de développement ordinaire.
user-invocable: false
---

Le développeur installe ork dans ce dépôt, ou veut revoir sa configuration. C'est un moment de mise en place : tu peux poser plusieurs questions, mais tu n'écris **rien** avant son accord.

$ARGUMENTS

# Les conventions d'ork

| Chemin | Rôle |
|---|---|
| `doc/ork.md` | configuration : frontmatter (`commande_tests`, `branche_integration`, `max_parallele`), puis les sections `## Plan requis quand` et `## Notes` |
| `doc/story/` | les stories, une par fichier |
| `doc/tickets/` | les tickets ouverts |
| `doc/tickets/captures/` | les captures d'écran jointes |
| `doc/tickets/done/` | les tickets terminés |

Ce qui est propre au projet ne va pas dans `doc/ork.md`, mais dans le `CLAUDE.md` du dépôt, que tous les ouvriers lisent : préparation d'un worktree, fichiers générés, vérifications supplémentaires, style de commit. `doc/ork.md` ne garde que la configuration d'ork et les écarts aux conventions.

# 1. Point de départ

- Si `doc/ork.md` existe, tu es en **révision** : lis-le, et ne propose ensuite que les écarts entre ce fichier et ce que tu détectes.
- Sinon, c'est une **installation**.
- Note la branche courante et `git status --short`. Ne touche pas aux modifications que tu y trouves.
- Lis `.claude/settings.json` et `.claude/settings.local.json` s'ils existent. Tu y cherches deux réglages :
  - `"agent": "ork:lead"` : le chef devient la session par défaut du projet ;
  - `"permissions": { "deny": ["Agent(fork)"] }` : un filet de sécurité contre les forks, qui héritent de toute la conversation du chef.

# 2. Enquête

Lance **un seul** `ork:scout`, avec ces questions. Garde sa conclusion, pas le détail.

1. **Commandes :** quelles commandes vérifient le projet (types, lint, tests, contrat ou code généré) ? Comment les lance la CI (`.github/workflows/`, etc.) ? Propose une `commande_tests` unique, enchaînée avec `&&`.
2. **Branche d'intégration :** sur quelle branche le travail quotidien est-il posé ? Indices : branche courante, branche par défaut du remote, branches de la CI, conventions écrites dans `CLAUDE.md` ou dans la doc.
3. **Dossiers :** quels dossiers existants jouent l'un des rôles du tableau ci-dessus sous un autre nom (par exemple `doc/stories/`, `docs/`, `tickets/`) ? Pour chacun, liste **toutes** ses références dans les fichiers suivis, sauf `doc/tickets/done/` (`git grep -n -I '<chemin>'`), en distinguant :
   - les chemins écrits en clair, qu'on peut remplacer mécaniquement ;
   - les chemins construits ou indirects (concaténation, glob, variable).
4. **`CLAUDE.md` :** décrit-il ces quatre points ? Pour chaque point absent, ce que tu as pu déduire du dépôt :
   - la préparation d'un worktree (dépendances, fichiers d'environnement non versionnés) ;
   - les fichiers générés à régénérer plutôt que fusionner (fichiers de verrouillage, contrats, clients) ;
   - les vérifications supplémentaires selon les zones touchées ;
   - le style de commit (à partir de `git log --format=%s -30`).
5. **Doublons :** des skills, commandes ou agents du projet (`.claude/skills/`, `.claude/commands/`, `.claude/agents/`) recoupent-ils ork (intégration de branches, tickets, suivi d'état, worktrees) ? Pour chacun, ce qu'il contient de propre au projet.
6. **Ressources partagées :** une ressource commune rend-elle dangereux deux ouvriers en parallèle ? Exemples : une base de test unique, un port fixe, un dossier de sortie partagé.
7. **Tickets dans git :** `doc/` est-il versionné, ou ignoré par `.gitignore` ?

# 3. Proposition

Présente tout en **un seul message**, en groupant :

- **La configuration proposée :** le frontmatter complet, puis les règles « Plan requis quand » adaptées au dépôt (modèle de données, API publique, auth, build, nombre de fichiers). Pour `max_parallele`, propose `1` la première semaine. Si une ressource partagée a été trouvée, dis pourquoi le parallélisme est risqué.
- **Les écarts de dossiers :** chemin actuel → convention, et le nombre de références à mettre à jour.
- **Ce qui manque à `CLAUDE.md` :** le texte que tu proposes d'y ajouter, dans une section du type « Git, worktrees et intégration ».
- **Les doublons :** ce qu'il faut en garder (dans `CLAUDE.md`) et ce que tu proposes de supprimer.
- **Les réglages recommandés** qui manquent, s'il y en a.

Puis fais trancher avec `AskUserQuestion`, en quatre questions au plus par appel :

- **Un dossier hors convention, par question :**
  - « Renommer et mettre à jour les références (Recommandé) », toujours en premier ;
  - « Garder le dossier, ajouter une note dans `doc/ork.md` ».

  Si des références sont construites ou indirectes, dis-le dans la description : elles ne se remplacent pas mécaniquement, et un ouvrier devra s'en charger dans un ticket.
- **Suivi git des tickets :** « Commités (Recommandé) », parce que l'historique des tickets suit le code et survit à un changement de machine ; ou « Ignorés par git ».
- **Les ajouts à `CLAUDE.md` et le sort des doublons**, si tu en as trouvé.
- **Les réglages recommandés**, s'il en manque : où les écrire ?
  - « `.claude/settings.local.json` (Recommandé) » : personnel, non commité ;
  - « `.claude/settings.json` » : commité, donc imposé à tous ceux qui ouvrent le dépôt.
- **La configuration**, si un choix reste ambigu (par exemple deux branches d'intégration plausibles).

# 4. Appliquer

Dans cet ordre, uniquement ce qui a été accepté.

1. **Renommer un dossier :**
   - `git mv <ancien> <nouveau>` si le dossier est suivi, `mv` sinon ;
   - remplace chaque chemin écrit en clair que l'enquête a trouvé. Ce sont des remplacements de texte exacts : ne modifie rien d'autre dans ces fichiers ;
   - **vérifie** avec `git grep -n -I --untracked '<ancien>' -- ':!doc/tickets/done'`. Le résultat doit être vide. Sinon, corrige si c'est un chemin en clair oublié ; si c'est une référence construite, laisse-la et crée un ticket `pret`, qui la liste, pour qu'un ouvrier s'en occupe ;
   - ne réécris jamais l'historique git, ni les tickets de `done/` : ce sont des archives.
2. **Garder un dossier :** ajoute dans `## Notes` de `doc/ork.md` une ligne comme « **Stories :** elles sont dans `doc/stories/`, pas dans `doc/story/` ».
3. **Écrire `doc/ork.md`.** En révision, ne change que ce qui a été accepté et garde le reste tel quel.
4. **Créer les dossiers** `doc/tickets/`, `doc/tickets/captures/` et `doc/tickets/done/`. S'ils sont commités, ajoute un `.gitkeep` dans chaque dossier vide. S'ils sont ignorés, ajoute `doc/tickets/` au `.gitignore`.
5. **Compléter `CLAUDE.md`** avec le texte accepté.
6. **Ajouter les réglages** acceptés dans le fichier choisi. Fusionne-les avec le contenu existant : ajoute à `permissions.deny` sans remplacer la liste, et garde les autres clés. Vérifie ensuite que le fichier reste du JSON valide.
7. **Traiter les doublons :** une fois leur contenu propre au projet passé dans `CLAUDE.md`, supprime ceux dont la suppression a été acceptée.

# 5. Commiter

Propose un commit de ces seuls changements, sur la branche courante, dans le style du dépôt. Laisse en dehors toute modification qui existait avant l'init. Ne pousse jamais. Si le développeur préfère commiter lui-même, donne-lui la liste des fichiers.

# 6. Rendre compte

En cinq lignes au plus :
- la configuration retenue ;
- les dossiers renommés et le nombre de références mises à jour (ou la note ajoutée) ;
- ce qui a été ajouté à `CLAUDE.md` et ce qui a été supprimé, ainsi que les réglages ajoutés ;
- les tickets créés, s'il y en a ;
- la suite : si tu as ajouté `agent`, redémarrer Claude Code pour qu'il s'applique ; `/reload-plugins` si tu as modifié des skills du projet ; puis `/ork:ticket` ou une story pour commencer.
