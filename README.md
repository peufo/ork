# ork

Un plugin Claude Code pour travailler avec **une seule session principale** qui orchestre des sous-agents. Tu lui parles, elle capture, découpe, lance, suit et te pose les questions au bon moment. Les sous-agents font le travail lourd dans leur propre worktree.

## Ce que contient le plugin

| Pièce | Rôle |
|---|---|
| `agents/lead.md` | Le chef d'orchestre (`ork:lead`). Lancé comme session principale, son prompt remplace celui de Claude Code et reste actif toute la session. |
| `agents/worker.md` | L'ouvrier (`ork:worker`) : exécute un ticket dans un worktree isolé, en mode plan ou exécution, et rend un rapport court au format fixe. |
| `agents/scout.md` | L'enquêteur (`ork:scout`) : enquête en lecture seule (sur Sonnet) pour trier une remarque ou trouver les zones touchées. |
| `skills/ticket` | `/ork:ticket` : capture rapide d'une remarque ou d'une capture d'écran. |
| `skills/status` | `/ork:status` : état de tous les tickets et livraison des notifications différées. |
| `skills/resume` | `/ork:resume` : redémarrer une session fraîche à partir des fichiers. |

Rien de tout ça ne va dans le dépôt du client. Dans le dépôt, il n'y a que des **données** en Markdown :

```
doc/
  ork.md                ← config du projet (commande de tests, branche, parallélisme)
  story/                ← tes stories, comme aujourd'hui
  tickets/
    T-0012-….md         ← un fichier par ticket ouvert, statut en frontmatter
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

Puis tape `/ork:resume`. Le premier jour, il te proposera de créer `doc/ork.md`.

Un alias rend ça confortable :

```bash
alias ork='claude --agent ork:lead'
```

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

Pour le mode de permission, le mode **auto** (une fois validé dans ton environnement) évite que chaque commande d'un ouvrier vienne te demander une approbation. Les sous-agents héritent du mode de la session principale. En mode manuel, les demandes d'autorisation des ouvriers en arrière-plan remontent dans la session du chef, avec le nom de l'ouvrier qui demande.

## Une journée type

- **Le matin** : `ork`, puis `/ork:resume`. Cinq lignes d'état, une question : par quoi on commence ?
- **Tu remarques un truc** : `/ork:ticket le bouton reste actif pendant l'envoi`, avec une capture collée. Tu reçois une ligne en retour et tu retournes à ce que tu faisais.
- **Idéation** : tu parles normalement d'une story. Le chef met à jour `doc/story/…`, propose un découpage en tickets et les crée après ton accord.
- **Les travaux partent seuls** : il décide de ce qui part en parallèle, selon les `zones` des tickets et `max_parallele`. Il lance le mode plan quand il le faut, et sinon l'exécution directe.
- **Il te dérange seulement pour** une question bloquante, un plan à valider, un échec ou un fait nouveau. Le reste attend ton `/ork:status`.
- **Relecture** : « montre-moi T-0012 » te donne le résumé, les endroits délicats et la commande de diff. Puis « intègre T-0012 » lance le merge, les tests et l'archivage.

## Le cycle d'un ticket

```
idee → pret → (plan → plan-a-valider →) en-cours → a-relire → fait
                                     ↘ bloque ↗
```

Un ouvrier qui rencontre une décision qui ne lui appartient pas **s'arrête** et rend `STATUT: BLOQUÉ` avec 2 ou 3 options et une recommandation. Un sous-agent ne peut pas te poser de question directement, c'est pour ça que ce protocole existe. Le chef te pose la question, puis relance le **même** ouvrier avec ta réponse. Celui-ci reprend avec tout son historique.

## Vue visuelle dans Obsidian

Chaque ticket a son statut en frontmatter, donc tu peux ouvrir `doc/` comme coffre Obsidian et afficher les tickets regroupés par `statut`, par exemple avec une vue *Bases* ou un plugin de tableau qui lit les propriétés. Je n'ai pas testé de configuration précise : à ajuster selon ce que tu utilises.

## Ce qu'il faut savoir (vérifié dans la doc Claude Code)

- **Le prompt du chef remplace entièrement celui de Claude Code**, comme `--system-prompt`. Ton `CLAUDE.md` est toujours chargé. C'est voulu : le chef n'a pas besoin des instructions de codage.
- **Les sous-agents sont en arrière-plan par défaut** : le chef reste disponible pendant qu'ils travaillent, et leurs rapports arrivent en notification.
- **Le worktree d'un ouvrier part de la branche par défaut du dépôt**, pas de la branche courante. L'ouvrier commence donc par s'aligner sur la `branche_integration` de `doc/ork.md`. Seul ce qui y est **commité** lui parvient : commite avant de lancer un ticket qui dépend d'un changement récent.
- **Les tickets ne sont écrits que par le chef**, dans le dépôt principal. Les ouvriers reçoivent le ticket recopié dans leur message, parce que leur worktree ne contient pas les fichiers non commités.
- **Les agents d'un plugin ignorent** les champs `permissionMode`, `hooks` et `mcpServers`. Si tu en as besoin plus tard, copie l'agent concerné dans `~/.claude/agents/`.
- **`initialPrompt` ne s'applique pas aux agents de plugin**, d'où le `/ork:resume` à taper au démarrage.
- **Le chef n'a pas de restriction d'outils**. La règle « je ne lis jamais de code » est une consigne, pas un verrou. Si tu veux la verrouiller, ajoute dans `lead.md` un champ `tools:` qui exclut `Edit` et `Write`. Il ne pourrait alors plus écrire les tickets lui-même : à tester si ça devient un problème.
- **Coût** : plusieurs ouvriers en parallèle consomment ta limite plus vite. `max_parallele: 2` est un bon départ.

## Démarrer petit

La première semaine, mets `max_parallele: 1` dans `doc/ork.md`. Le but est de roder trois choses : le format des rapports, le protocole de blocage et le seuil « plan requis ». Tu ajusteras ensuite les prompts de `agents/` selon ce qui t'agace. Ce sont des fichiers Markdown, et `/reload-plugins` recharge les changements sans redémarrer.
