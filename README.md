# ork

Un plugin pour l'onglet Code de l'app Claude. Tu parles à **une seule conversation**, le chef : elle capture tes remarques, découpe, lance des ouvriers en sous-agents, suit leur travail et te pose les questions au bon moment. Les ouvriers font le travail lourd dans leur propre worktree.

ork ne refait pas ce que l'app sait déjà faire (terminal, navigateur, fichiers, diff, tâches). Il ajoute ce qui lui manque :

- **le pilotage** : un sujet à la fois, des niveaux de décision, le parallélisme selon les zones touchées et les dépendances ;
- **les tickets** : de simples fichiers Markdown, qui survivent à la compaction et aux redémarrages ;
- **l'intégration** : rebase et tests dans le worktree, puis avance rapide sur ta branche, sans jamais pousser.

## Ce que contient le plugin

| Pièce | Rôle |
|---|---|
| `output-styles/ork.md` | Le rôle du chef. Il s'ajoute au prompt par défaut de Claude Code, sans le remplacer. |
| `agents/worker.md` | L'ouvrier : exécute un ticket dans un worktree isolé et rend un rapport court au format fixe. |
| `agents/critic.md` | Le contre-lecteur : relit une story à froid et rend trois objections au plus. |
| `skills/` | Les procédures, chargées à la demande : `init`, `status`, `ticket`, `ideation`, `dispatch`, `review`, `integrate`, `stop`, `ui`. |
| `hooks/` | Le mod d'affichage : badges, carte de sujet, boutons et panneau d'état. |
| `app/`, `scripts/ork-app.sh` | La micro-app, un bonus facultatif (voir plus bas). |

Dans le dépôt de ton projet, il n'y a que des données en Markdown :

```
.ork/
  config.md             ← dossier de doc, commande de tests, branche, parallélisme, revue, autonomie
  tickets/
    t12-….md            ← un fichier par ticket ouvert, statut en frontmatter
    captures/           ← captures d'écran jointes aux tickets
    done/               ← tickets intégrés
doc/                    ← ta documentation (docs_dir dans la config)
  story/                ← tes stories
```

## Installation

Clone le dépôt, puis relie-le à ton dossier de skills personnel. Claude Code charge tout seul un dossier de `~/.claude/skills/` qui contient un `.claude-plugin/plugin.json` :

```bash
mkdir -p ~/.claude/skills
```

```bash
ln -s "$PWD" ~/.claude/skills/ork
```

## Démarrer dans un projet

1. Ouvre une session de l'app sur le dossier du projet, puis tape `/ork:init`. Il détecte les commandes et la branche, puis propose `.ork/config.md` et le réglage du chef.
2. Accepte le réglage, écrit dans `.claude/settings.local.json` :
   ```json
   { "outputStyle": "ork:ork", "permissions": { "deny": ["Agent(fork)"] } }
   ```
3. Ouvre une nouvelle session **sans worktree**. Le chef écrit dans `.ork/tickets/` et intègre dans le dépôt principal : il doit y travailler directement.

Les ouvriers et le contre-lecteur gardent leur propre prompt. En revanche, toutes les sessions principales de ce projet deviennent le chef tant que ce réglage est posé.

Pour le mode de permission, le mode **Auto** évite que chaque commande d'un ouvrier te demande une approbation.

## Une journée type

- **Le matin** : `/ork:status` signale les ouvriers arrêtés et les worktrees à surveiller, puis présente un seul sujet. Le panneau montre le reste.
- **Tu remarques un truc** : écris-le, avec une capture si tu veux. Tu reçois une ligne en retour et tu retournes à ce que tu faisais.
- **Idéation** : tu parles normalement d'une story. Le chef l'écrit dans `<docs_dir>/story/`, propose un découpage et crée les tickets après ton accord.
- **Les travaux partent seuls** : il décide de ce qui part en parallèle, selon les `zones`, `depends_on` et `max_parallel`.
- **Il te dérange seulement pour** une question bloquante, un plan à valider, un échec ou un fait nouveau.
- **Il te demande trop, ou pas assez ?** Dis-le. Il propose une règle d'une ligne pour `## Ask before` ou `## No need to ask`, et l'écrit après ton accord. Push, force, dépôt principal sale et validation des plans restent hors dosage.
- **Revue automatique** : avec `auto_review: [/simplify, /code-review medium --fix]`, chaque ouvrier passe ces revues avant de commiter. Avec `review: per-ticket`, le chef intègre seul un ticket sans plan dont la revue est propre. `review: required` dans un ticket force ta relecture.
- **Note de confiance** : chaque point de relecture et chaque recommandation d'un ouvrier porte une note sur 10. `confidence_threshold: 8` laisse passer ce qui atteint 8, signalé par `[AUTO]`. Absent : tu vois tout. Les plans, `review: required` et `## Ask before` passent toujours par toi.
- **Relecture guidée** : « relisons t12 » donne une vue d'ensemble triée par note, puis chaque point à relire. Le chef ouvre le diff ou le fichier dans les panneaux de l'app.
- **Intégration** : « intègre t12 » rebase la branche et la vérifie dans son worktree, la pose par avance rapide, puis archive le ticket et supprime le worktree.
- **Le soir, ou avant de fermer** : dis « arrête-toi au prochain état stable », ou clique sur `Arrêt propre` dans le panneau. Le chef ne lance plus rien, les ouvriers commitent sur leur branche et rendent la main, puis il te dit que tu peux fermer et ce qu'il faudra reprendre. Ce qui n'est pas intégré attend sur sa branche.
- **Conventions du projet** : ce qui est propre au projet va dans `CLAUDE.md`, que tous les ouvriers lisent : préparation d'un worktree, fichiers générés, vérifications, style de commit.

## Le cycle d'un ticket

```
idea → ready → (planning → plan-review →) in-progress → in-review → done
                                     ↘ blocked ↗
```

Un ouvrier qui rencontre une décision qui ne lui appartient pas **s'arrête** et rend `STATUS: BLOCKED` avec ses options et une recommandation. Le chef te pose la question, puis relance le **même** ouvrier avec ta réponse.

## Ce que dessine le mod

Le chef écrit du texte au même format, et le mod l'affiche sans modifier le message enregistré. Le panneau, lui, lit directement les tickets :

- les lignes `[TAG]` deviennent des badges colorés ;
- un sujet devient une carte, avec son lien, son étape et ses réponses en boutons. Une fois ta réponse envoyée, les boutons disparaissent, pour qu'un clic tardif ne réponde pas au mauvais sujet ;
- un panneau latéral montre en permanence ce qui t'attend, ce qui tourne, ce qui est prêt et les idées, avec un bouton par ticket, le ticket du sujet en cours marqué d'un point, et un loader par agent au travail, avec ce qu'il fait. Le chef n'a donc plus à répéter cet état. Il s'ouvre au démarrage dans un projet ork, et `/ork` le rouvre.

Si le mod ne reconnaît pas un texte, ou s'il cesse de fonctionner, le texte s'affiche normalement. Son API est en accès anticipé et peut changer d'une version de Claude Code à l'autre.

## Bonus : la micro-app

Facultative : rien dans le pilotage n'en dépend. `/ork:ui` ouvre dans le navigateur de l'app une page locale, propre au dépôt :

- **Tickets** : un kanban, une colonne par statut, mis à jour tout seul quand le chef écrit un ticket.
- **Agents** : sur la carte d'un ticket, un bouton par agent au travail, avec son fil d'activité en direct. Un agent sans ticket apparaît dans l'en-tête.
- **Documents** : tous les Markdown de `.ork/` et de `docs_dir`, avec leurs liens et les captures citées dans les tickets.

Elle ne fait que lire et n'écoute que sur `127.0.0.1`. Le fil d'activité lit les transcripts sous `~/.claude/projects/<dépôt>/`. La doc de Claude Code ne garantit pas leur format : si une mise à jour le change, l'agent reste affiché, sans son détail. Les sessions lancées depuis un worktree ne sont pas lues.

Au premier usage d'une version d'ork, `scripts/ork-app.sh` télécharge le binaire de ta plateforme (macOS ou Linux, arm64 ou amd64) depuis les releases GitHub, vérifie sa somme de contrôle et le garde dans `~/.cache/ork/`. Cette somme vit dans la même release : elle protège d'un téléchargement corrompu, pas d'une release compromise. Ses seules traces dans le projet sont `ork-app.json` et `ork-app.log`, dans le dossier `.git`.

## Développer

**Le mod**, depuis la racine :

```bash
claude plugin validate . && claude plugin test .
```

**La micro-app** (il faut Go et `golangci-lint`), depuis `app/` :

```bash
golangci-lint run && go test -race ./...
```

```bash
sh run.sh <projet>
```

**Publier une version** : passe `version` à la nouvelle valeur dans `.claude-plugin/plugin.json`, commite, puis pousse le tag `v<version>`. Le workflow `release.yml` publie les binaires. Entre le commit et la fin du workflow, `/ork:ui` ne trouve pas encore le binaire et le dit.

## Ce qu'il faut savoir

- **Le chef n'a pas de verrou** : « je ne lis jamais de code » est une consigne. Si tu lui demandes de modifier une ligne, il le fait et te signale la règle contournée.
- **Seul le dernier message d'un tour s'affiche tel quel.** L'app résume le texte écrit avant un appel d'outil, c'est pourquoi le chef écrit ses lignes d'état à la fin du tour.
- **Le worktree d'un ouvrier part de la branche par défaut du dépôt.** L'ouvrier s'aligne d'abord sur `integration_branch`, et seul ce qui y est **commité** lui parvient.
- **Les tickets ne sont écrits que par le chef**, dans le dépôt principal. Les ouvriers reçoivent le ticket recopié dans leur message.
- **Les agents d'un plugin ignorent** les champs `permissionMode`, `hooks` et `mcpServers`.
- **Coût** : plusieurs ouvriers en parallèle consomment ta limite plus vite. Un ouvrier coupé par la limite ne reprend pas de lui-même : le chef le signale et propose de le reprendre, avec tout son historique.

La première semaine, mets `max_parallel: 1` dans `.ork/config.md`, le temps de roder le format des rapports, le protocole de blocage et le seuil « plan requis ».
