---
name: init
description: Configurer ork dans un dépôt ou revoir sa configuration. Détecte les commandes et la branche d'intégration, aligne les dossiers sur les conventions d'ork, vérifie que CLAUDE.md donne aux ouvriers ce dont ils ont besoin, puis écrit `.ork/config.md` après accord.
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

`.ork/` appartient à ork. La documentation du projet, stories comprises, reste dans son propre dossier : `docs_dir`, `doc/` par défaut.

Ce qui est propre au projet ne va pas dans `.ork/config.md`, mais dans le `CLAUDE.md` du dépôt, que tous les ouvriers lisent : préparation d'un worktree, fichiers générés, vérifications supplémentaires, style de commit. `.ork/config.md` ne garde que la configuration d'ork et les écarts aux conventions.

# Le fichier `.ork/config.md`

```markdown
---
docs_dir: doc/
test_command: npm run check && npm test
integration_branch: main
max_parallel: 2
auto_review: [/simplify, /code-review medium --fix]
review: per-ticket
confidence_threshold: 8
---

## Plan required when
- le ticket touche le modèle de données, une API publique ou la configuration de build ;
- le ticket touchera probablement plus de 5 fichiers.

## Review required when
- le ticket touche l'authentification ou une migration ;
- le diff supprime ou transforme des données existantes.

## Ask before
- lancer un ticket qui touche `src/lib/server/`.

## No need to ask
- relancer une fois, un cran au-dessus, un ticket en `FAILED` sur des tests rouges.

## Notes
Tout ce qui est utile à tous les ouvriers et qui n'est pas déjà dans CLAUDE.md.
```

- `docs_dir` : le dossier de documentation du projet, qui contient les stories. Absent : `doc/`. C'est un sous-dossier du dépôt, non caché.
- `auto_review` : les commandes de revue que l'ouvrier lance avant de commiter. Absent ou vide : pas de revue auto.
- `review` : `always` (défaut) ou `per-ticket`, qui permet au chef d'intégrer seul un ticket sans plan dont la revue est propre.
- `confidence_threshold` : ce qui atteint cette note passe sans le développeur. Absent : tout est montré. `0` : tout passe.

# 1. Point de départ

- Si `.ork/config.md` existe, tu es en **révision** : lis-le, et ne propose ensuite que les écarts entre ce fichier et ce que tu détectes.
- Sinon, c'est une **installation**.
- Note la branche courante et `git status --short`. Ne touche pas aux modifications que tu y trouves.
- Lis `.claude/settings.json` et `.claude/settings.local.json` s'ils existent. Tu y cherches ce réglage :
  - `"permissions": { "deny": ["Agent(fork)"] }` : un filet de sécurité contre les forks, qui héritent de toute la conversation du chef ;

# 2. Enquête

Lance **un seul** agent `Explore` (modèle `sonnet`, sans rien modifier), avec ces questions. Garde sa conclusion, pas le détail.

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

# 3. Proposition

Présente tout en **un seul message**, en groupant :

- **La configuration proposée :** le frontmatter complet, puis les règles « Plan required when » adaptées au dépôt (modèle de données, API publique, auth, build, nombre de fichiers). Pour `max_parallel`, propose `1` la première semaine. Si une ressource partagée a été trouvée, dis pourquoi le parallélisme est risqué. Pour la revue, propose `auto_review: [/simplify]` et `review: always` la première semaine : le développeur juge la qualité de la revue auto avant de passer à `per-ticket`. Pour la même raison, laisse `confidence_threshold` absent la première semaine : le développeur voit tous les points avec leur note, et juge à partir de quelle note il peut les laisser passer. Propose des règles `## Review required when` à partir des zones sensibles trouvées. Laisse `## Ask before` et `## No need to ask` vides : elles se remplissent avec les corrections du développeur.
- **Les écarts de dossiers :** chemin actuel → convention, et le nombre de références à mettre à jour.
- **Ce qui manque à `CLAUDE.md` :** le texte que tu proposes d'y ajouter, dans une section du type « Git, worktrees et intégration ».
- **Les doublons :** ce qu'il faut en garder (dans `CLAUDE.md`) et ce que tu proposes de supprimer.
- **Les réglages recommandés** qui manquent, s'il y en a.

Puis fais trancher avec `AskUserQuestion`, en quatre questions au plus par appel :

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

Propose un commit de ces seuls changements, sur la branche courante, dans le style du dépôt. Laisse en dehors toute modification qui existait avant l'init. Ne pousse jamais. Si le développeur préfère commiter lui-même, donne-lui la liste des fichiers.

# 6. Rendre compte

En six lignes au plus :
- la configuration retenue ;
- les dossiers renommés et le nombre de références mises à jour (ou la note ajoutée) ;
- ce qui a été ajouté à `CLAUDE.md` et ce qui a été supprimé, ainsi que les réglages ajoutés ;
- les tickets créés, s'il y en a ;
- la suite : dans une session sans worktree, taper `/ork` pour démarrer le chef, puis faire une remarque ou parler d'une story.
