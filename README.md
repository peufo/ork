# ork

Un plugin Claude Code pour travailler avec **une seule session principale** qui orchestre des sous-agents. Tu lui parles, elle capture, découpe, lance, suit et te pose les questions au bon moment. Les sous-agents font le travail lourd dans leur propre worktree.

## Ce que contient le plugin

| Pièce | Rôle |
|---|---|
| `agents/lead.md` | Le chef d'orchestre (`ork:lead`). Lancé comme session principale, son prompt remplace celui de Claude Code et reste actif toute la session. |
| `agents/worker.md` | L'ouvrier (`ork:worker`) : exécute un ticket dans un worktree isolé, en mode plan ou exécution, et rend un rapport court au format fixe. |
| `agents/scout.md` | L'enquêteur (`ork:scout`) : enquête en lecture seule (sur Sonnet) pour trier une remarque ou trouver les zones touchées. |
| `agents/critic.md` | Le contre-lecteur (`ork:critic`) : relit une story à froid, sans la conversation, et rend trois objections au plus. |
| `skills/init` | `/ork:init` : installer ork dans un dépôt ou revoir sa configuration. Il détecte les commandes, aligne les dossiers sur les conventions (ou ajoute une note) et complète `CLAUDE.md` pour les ouvriers. |
| `skills/ideation` | Idéation sur une story : chargé par le chef dès qu'on discute d'une fonctionnalité, ou par `/ork:ideation`. Il challenge les idées sous plusieurs angles et fait des contre-propositions quand elles sont meilleures. |
| `skills/ticket` | `/ork:ticket` : capture rapide d'une remarque ou d'une capture d'écran. |
| `skills/status` | `/ork:status` : état de tous les tickets, au démarrage d'une session comme en cours de journée, et livraison des notifications différées. |
| `skills/ui` | `/ork:ui` : ouvre la micro-app, un kanban des tickets et un lecteur des Markdown de `doc/`, en lecture seule. |
| `app/`, `scripts/ork-app.sh` | Le code de la micro-app (un binaire Go) et le lanceur qui télécharge le binaire au premier usage. |

Rien de tout ça ne va dans le dépôt du client. Dans le dépôt, il n'y a que des **données** en Markdown :

```
doc/
  ork.md                ← config du projet (commande de tests, branche, parallélisme, revue, autonomie)
  story/                ← tes stories, comme aujourd'hui
  tickets/
    t12-….md         ← un fichier par ticket ouvert, statut en frontmatter
    captures/           ← captures d'écran jointes aux tickets
    done/               ← tickets intégrés
```

## Installation

Copie le dossier `ork/` (celui qui contient `.claude-plugin/`) dans ton dossier de skills personnel :

```bash
mkdir -p ~/.claude/skills
cp -r ork ~/.claude/skills/ork
```

Claude Code charge automatiquement, dans toutes les sessions, un dossier de `~/.claude/skills/` qui contient un `.claude-plugin/plugin.json`. Il n'y a ni marketplace ni étape d'installation. Vérifie avec :

```bash
claude plugin validate ~/.claude/skills/ork
claude plugin list
```

## Démarrer

Dans le dépôt du projet :

```bash
claude --agent ork:lead
```

Le premier jour, tape `/ork:init` : il crée `doc/ork.md` et `doc/tickets/` après ton accord. Les jours suivants, tape `/ork:status`.

Le chef ne présente qu'**un sujet à la fois**, le plus urgent, et ta réponse porte toujours sur celui-là. Ses messages commencent par des lignes d'état, une par fait, qui se lisent d'un coup d'œil :

```
[OK] `t21-nom-deck-langue-automatique` -> `dev`
[HINT] `bun run sandbox:reset` pour voir le deck Anglais
[DEP] `t22-remote-functions-auth-oauth` -> `t27-colonnes-state-before`
[WAIT] 👀 `t28-traduire-langue-apprise-vers-connue`

### 👀 [t27-colonnes-state-before](doc/tickets/t27-colonnes-state-before.md) · 2 à relire sur 4
Remplace le JSON `state_before` de `review_logs` par 9 colonnes typées.
6 fichiers · +182 −40 · 3 commits

1. [5/10] `migration.sql:17` : casts vers `TIMESTAMP(3)` et `review_state`
2. [6/10] `decks.md:25` : la story cite encore `ReviewLog.stateBefore`
3. [8/10] ✓ `schema.ts:40` : noms des colonnes, validés au plan
4. [9/10] ✓ `amend.test.ts:1` : nouveau test `amend`

`relis` : commencer par le point 1
`sujet suivant` : passer à `t28-traduire-langue-apprise-vers-connue`
```

Vient ensuite le sujet en cours : l'icône de sa nature, le ticket (un lien vers son fichier) et l'étape (par exemple `point 2/4` d'une relecture), un corps de quelques lignes, puis une ligne par réponse attendue. Les autres sujets en attente sont les lignes `[WAIT]`. Pour passer à un autre sujet, dis « sujet suivant ». Le vocabulaire complet est dans « Les lignes d'état » de `agents/lead.md`.

Un alias rend ça confortable :

```bash
alias ork='claude --agent ork:lead --name ork'
```

`--name` donne un nom à la session : il s'affiche dans le titre de l'onglet du terminal, dans la zone de saisie et dans `/resume`. Tu repères ainsi l'onglet du chef parmi les autres sessions.

Tu peux aussi en faire le comportement par défaut d'un projet sans rien commiter, avec `.claude/settings.local.json` :

```json
{ "agent": "ork:lead" }
```

### Réglages recommandés

Toujours dans `.claude/settings.local.json` du projet, ou dans `~/.claude/settings.json` :

```json
{
  "permissions": {
    "deny": ["Agent(fork)"]
  }
}
```

Ce réglage empêche Claude de « forker » la conversation, car un fork hérite de toute la conversation de l'orchestrateur. Le prompt du chef l'interdit déjà, ceci n'est qu'un filet de sécurité.

Le chef peut reprendre seul après la limite d'usage de ton abonnement. Quand la limite tombe, Claude Code affiche un menu : choisis d'attendre le reset, et la session reprend la tâche interrompue. Ce comportement dépend du réglage **« Continue automatically at usage limit »** du `/config` du terminal (absent des réglages de l'extension VS Code). Ce réglage est actif par défaut. Ce n'est pas un réglage du plugin. Il faut que la session reste ouverte (pas de terminal fermé ni de Mac en veille) et qu'elle soit connectée avec un abonnement claude.ai, pas avec une clé API. Une demande d'autorisation peut encore la mettre en pause, et Échap annule l'attente.

Pour le mode de permission, le mode **auto** (une fois validé dans ton environnement) évite que chaque commande d'un ouvrier vienne te demander une approbation. Les sous-agents héritent du mode de la session principale. En mode manuel, les demandes d'autorisation des ouvriers en arrière-plan remontent dans la session du chef, avec le nom de l'ouvrier qui demande.

## Une journée type

- **Le matin** : `ork`, puis `/ork:status`. L'état des tickets (ceux qui tournaient hier sont signalés comme arrêtés), puis une seule question ou proposition.
- **Tu remarques un truc** : `/ork:ticket le bouton reste actif pendant l'envoi`, avec une capture collée. Tu reçois une ligne en retour et tu retournes à ce que tu faisais.
- **Idéation** : tu parles normalement d'une story. Le chef met à jour `doc/story/…`, propose un découpage en tickets et les crée après ton accord.
- **Les travaux partent seuls** : il décide de ce qui part en parallèle, selon les `zones` des tickets et `max_parallel`. Un ticket dont le `depends_on` cite un ticket pas encore intégré attend, puis part seul dès que ce dernier l'est. Il lance le mode plan quand il le faut, et sinon l'exécution directe.
- **Il te dérange seulement pour** une question bloquante, un plan à valider, un échec ou un fait nouveau. Le reste attend ton `/ork:status`.
- **Il te demande trop, ou pas assez ?** Dis-le (« pas besoin de me demander ça », « demande-moi avant »). Il propose une règle d'une ligne pour `## Ask before` ou `## No need to ask` dans `doc/ork.md`, et l'écrit après ton accord. Le tableau « Qui décide quoi » de `agents/lead.md` donne les niveaux par défaut. Push, force, dépôt principal sale et validation des plans restent hors dosage.
- **Revue automatique** : avec `auto_review: [/simplify, /code-review medium --fix]` dans `doc/ork.md`, chaque ouvrier passe ces revues sur son travail avant de commiter. Avec `review: per-ticket`, le chef intègre seul un ticket sans plan dont la revue auto est propre et où l'ouvrier ne signale rien. `review: required` dans un ticket force ta relecture. Le chef le pose seul quand une règle de `## Review required when` s'applique (auth, migrations…).
- **Note de confiance** : chaque point de relecture et chaque recommandation d'un ouvrier porte une note sur 10 (0 : aucune idée, 10 : rien ne manque, certain). `confidence_threshold: 8` dans `doc/ork.md` laisse passer ce qui atteint 8 : le point est coché d'office, la question est tranchée par la recommandation et signalée par une ligne `[AUTO]`. Absent : tu vois tout ; `0` : tout passe. Les plans, `review: required` et `## Ask before` passent toujours par toi.
- **Relecture guidée** : « relisons t12 » te donne une vue d'ensemble (ce que fait le ticket, le volume du diff, tous les points triés par note, ceux cochés d'office marqués `✓`), puis chaque point à relire, un par message, en ouvrant la diff du fichier dans VS Code (`code -d`, avant / après). Tu réponds « ok » ou une remarque, que le chef transmet à l'ouvrier. Puis « intègre t12 » rebase la branche sur la branche d'intégration et la vérifie dans son worktree (c'est l'ouvrier qui le fait, en INTEGRATION MODE). Le chef la pose ensuite par avance rapide (`merge --ff-only`), puis archive le ticket et supprime le worktree. L'historique reste linéaire, et la branche d'intégration ne bouge que lorsque tout est vert.
- **Worktree hors ticket** : « intègre <worktree> » fonctionne aussi pour une session lancée à part. `/ork:status` signale ces worktrees orphelins.
- **Conventions du projet** : ork reste générique. Ce qui est propre au projet va dans le `CLAUDE.md` du dépôt, que tous les ouvriers lisent : préparation d'un worktree, fichiers générés à régénérer plutôt que fusionner, vérifications supplémentaires selon les zones, style de commit.

## Le cycle d'un ticket

```
idea → ready → (planning → plan-review →) in-progress → in-review → done
                                     ↘ blocked ↗
```

Un ouvrier qui rencontre une décision qui ne lui appartient pas **s'arrête** et rend `STATUS: BLOCKED` avec 2 ou 3 options et une recommandation. Un sous-agent ne peut pas te poser de question directement, c'est pour ça que ce protocole existe. Le chef te pose la question, puis relance le **même** ouvrier avec ta réponse. Celui-ci reprend avec tout son historique.

## La micro-app

`/ork:ui` ouvre dans ton navigateur une page locale, propre au dépôt :

- **Tickets** : un kanban, une colonne par statut. Il se met à jour tout seul quand le chef écrit un ticket, et une carte qui vient de changer de colonne se signale brièvement. `blocked`, `plan-review` et `in-review` ressortent : ce sont eux qui t'attendent.
- **Documents** : tous les Markdown de `doc/`, avec leurs liens, et les captures d'écran citées dans les tickets.

Elle ne fait que lire : elle n'a aucune route d'écriture, et le chef reste le seul à écrire dans `doc/tickets/`. Elle écoute seulement sur `127.0.0.1`.

**Rien à installer.** Au premier usage d'une version d'ork, `scripts/ork-app.sh` télécharge le binaire de ta plateforme (macOS ou Linux, arm64 ou amd64) depuis les releases GitHub, vérifie sa somme de contrôle et le garde dans `~/.cache/ork/`. Cette somme vit dans la même release : elle protège d'un téléchargement corrompu, pas d'une release compromise. Le serveur tourne en arrière-plan, et ses seules traces dans le projet sont `ork-app.json` et `ork-app.log` dans le dossier `.git`.

Hors de Claude Code, depuis le dépôt d'un projet :

```bash
alias ork-ui='sh ~/.claude/skills/ork/scripts/ork-app.sh --detach --open'
sh ~/.claude/skills/ork/scripts/ork-app.sh --stop    # pour l'arrêter
```

**Pour modifier la micro-app** (il faut Go et `golangci-lint`), depuis `app/` :

```bash
golangci-lint run && go test -race ./...
go build -o /tmp/ork-app .
ORK_APP_BIN=/tmp/ork-app sh ../scripts/ork-app.sh --repo <projet> --open
```

**Pour publier une version** : passe `version` à la nouvelle valeur dans `.claude-plugin/plugin.json`, commite, puis pousse le tag `v<version>` correspondant. Le workflow `release.yml` vérifie le tout et publie les binaires. Entre le commit et la fin du workflow, `/ork:ui` ne trouve pas encore le binaire et le dit.

## Ce qu'il faut savoir (vérifié dans la doc Claude Code)

- **Le prompt du chef remplace entièrement celui de Claude Code**, comme `--system-prompt`. Ton `CLAUDE.md` est toujours chargé. C'est voulu : le chef n'a pas besoin des instructions de codage. Les règles générales qui lui servent (outils, contenu non fiable, actions irréversibles, compte rendu fidèle) sont reprises dans la section « Socle » de `lead.md`. Les ouvriers, comme tout sous-agent, n'ont jamais reçu le prompt par défaut : la section « Socle » de `worker.md` en reprend l'essentiel pour coder.
- **Les sous-agents sont en arrière-plan par défaut** : le chef reste disponible pendant qu'ils travaillent, et leurs rapports arrivent en notification.
- **Le worktree d'un ouvrier part de la branche par défaut du dépôt**, pas de la branche courante. L'ouvrier commence donc par s'aligner sur la `integration_branch` de `doc/ork.md`. Seul ce qui y est **commité** lui parvient : commite avant de lancer un ticket qui dépend d'un changement récent. Entre deux tickets, c'est `depends_on` qui s'en charge : le chef attend que le premier soit intégré.
- **Les tickets ne sont écrits que par le chef**, dans le dépôt principal. Les ouvriers reçoivent le ticket recopié dans leur message, parce que leur worktree ne contient pas les fichiers non commités.
- **Les agents d'un plugin ignorent** les champs `permissionMode`, `hooks` et `mcpServers`. Si tu en as besoin plus tard, copie l'agent concerné dans `~/.claude/agents/`.
- **`initialPrompt` ne s'applique pas aux agents de plugin**, d'où le `/ork:status` à taper au démarrage.
- **Le chef n'a pas de restriction d'outils**. La règle « je ne lis jamais de code » est une consigne, pas un verrou. Si tu veux la verrouiller, ajoute dans `lead.md` un champ `tools:` qui exclut `Edit` et `Write`. Il ne pourrait alors plus écrire les tickets lui-même : à tester si ça devient un problème.
- **Coût** : plusieurs ouvriers en parallèle consomment ta limite plus vite. `max_parallel: 2` est un bon départ. Un ouvrier coupé par la limite ne reprend pas de lui-même. Le chef le signale et propose de le reprendre par `SendMessage`, avec tout son historique.

## Démarrer petit

La première semaine, mets `max_parallel: 1` dans `doc/ork.md`. Le but est de roder trois choses : le format des rapports, le protocole de blocage et le seuil « plan requis ». Tu ajusteras ensuite les prompts de `agents/` selon ce qui t'agace. Ce sont des fichiers Markdown, et `/reload-plugins` recharge les changements sans redémarrer.
