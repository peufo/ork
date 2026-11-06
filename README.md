# ork

Un plugin pour l'onglet Code de l'app Claude. Tu parles à **une seule conversation**, le chef : elle capture tes remarques, découpe, lance des ouvriers en sous-agents, suit leur travail et te pose les questions au bon moment. Les ouvriers font le travail lourd dans leur propre worktree.

ork ne refait pas ce que l'app sait déjà faire (terminal, navigateur, fichiers, diff, tâches). Il ajoute ce qui lui manque :

- **le pilotage** : un sujet à la fois, des niveaux de décision, le parallélisme selon les zones touchées et les dépendances ;
- **les tickets** : de simples fichiers Markdown, qui survivent à la compaction et aux redémarrages ;
- **l'intégration** : rebase et tests dans le worktree, puis avance rapide sur ta branche, sans jamais pousser.

## Ce que contient le plugin

| Pièce | Rôle |
|---|---|
| `agents/worker.md` | L'ouvrier : exécute un ticket dans un worktree isolé, tient ce ticket à jour et rend un rapport très court. |
| `agents/critic.md` | Le contre-lecteur : relit une story à froid et rend trois objections au plus. |
| `skills/start` | Le rôle du chef, chargé par `/ork` pour la session en cours. |
| `skills/` | Les procédures, chargées à la demande : `init`, `status`, `ticket`, `ideation`, `dispatch`, `review`, `integrate`, `stop`. |
| `hooks/` | Le mod d'affichage : badges, carte de sujet, boutons et panneau d'état. |

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

1. Ouvre une session de l'app sur le dossier du projet, puis tape `/ork:init`. Il détecte les commandes et la branche, puis propose `.ork/config.md`.
2. Dans une session **sans worktree**, tape `/ork`. Le panneau s'ouvre, cette conversation devient le chef et elle fait le point. Le chef écrit dans `.ork/tickets/` et intègre dans le dépôt principal : il doit y travailler directement.
3. Pour finir, tape `/ork:stop`. Le chef laisse tout se poser, te dit quand fermer, et la conversation redevient normale.

Rien n'est réglé au niveau du dépôt : les autres sessions du projet restent des sessions normales. Les ouvriers et le contre-lecteur gardent leur propre prompt.

Pour le mode de permission, le mode **Auto** évite que chaque commande d'un ouvrier te demande une approbation.

## Une journée type

- **Le matin** : `/ork` signale les ouvriers arrêtés et les worktrees à surveiller, puis présente un seul sujet. Le panneau montre le reste.
- **Tu remarques un truc** : écris-le, avec une capture si tu veux. Tu reçois une ligne en retour et tu retournes à ce que tu faisais.
- **Idéation** : tu parles normalement d'une story. Le chef l'écrit dans `<docs_dir>/story/`, propose un découpage et crée les tickets après ton accord.
- **Les travaux partent seuls** : il décide de ce qui part en parallèle, selon les `zones`, `depends_on` et `max_parallel`.
- **Il te dérange seulement pour** une question bloquante, un plan à valider, un échec ou un fait nouveau.
- **Il te demande trop, ou pas assez ?** Dis-le. Il propose une règle d'une ligne pour `## Ask before` ou `## No need to ask`, et l'écrit après ton accord. Push, force, dépôt principal sale et validation des plans restent hors dosage.
- **Revue automatique** : avec `auto_review: [/simplify, /code-review medium --fix]`, chaque ouvrier passe ces revues avant de commiter. Avec `review: per-ticket`, le chef intègre seul un ticket sans plan dont la revue est propre. `review: required` dans un ticket force ta relecture.
- **Note de confiance** : chaque point de relecture et chaque recommandation d'un ouvrier porte une note sur 10. `confidence_threshold: 8` laisse passer ce qui atteint 8, signalé par `[AUTO]`. Absent : tu vois tout. Les plans, `review: required` et `## Ask before` passent toujours par toi.
- **Relecture guidée** : « relisons t12 » place le dépôt principal sur une branche `review/t12` et ouvre le panneau de modifications de l'app, puis donne une vue d'ensemble triée par note et chaque point à relire. Une seule relecture à la fois : rien ne se pose sur la branche d'intégration avant la fin de celle-ci.
- **Intégration** : « intègre t12 » rebase la branche et la vérifie dans son worktree, la pose par avance rapide, puis archive le ticket et supprime le worktree.
- **Le soir, ou avant de fermer** : tape `/ork:stop`, dis « arrête-toi au prochain état stable », ou clique sur `Arrêt propre` dans le panneau. Le chef ne lance plus rien, les ouvriers commitent sur leur branche et rendent la main, puis il te dit que tu peux fermer et ce qu'il faudra reprendre. Ce qui n'est pas intégré attend sur sa branche.
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
- un panneau latéral montre en permanence ce qui t'attend, ce qui tourne, ce qui est prêt et les idées, avec un bouton par ticket, le ticket du sujet en cours marqué d'un point, et un loader par agent au travail, avec ce qu'il fait. Le chef n'a donc plus à répéter cet état. `/ork` l'ouvre.

Si le mod ne reconnaît pas un texte, ou s'il cesse de fonctionner, le texte s'affiche normalement. Son API est en accès anticipé et peut changer d'une version de Claude Code à l'autre.

## Développer

Depuis la racine :

```bash
claude plugin validate . && claude plugin test .
```

**Publier une version** : passe `version` à la nouvelle valeur dans `.claude-plugin/plugin.json`, commite, puis pousse le tag `v<version>`.

## Ce qu'il faut savoir

- **Le rôle du chef vit dans la conversation.** Il s'ajoute au prompt par défaut, consignes de codage comprises. Une compaction peut couper la fin du skill : le chef le recharge alors, et `/ork` le relance en cas de doute.
- **Le chef n'a pas de verrou** : « je ne lis jamais de code » est une consigne. Si tu lui demandes de modifier une ligne, il le fait et te signale la règle contournée.
- **Seul le dernier message d'un tour s'affiche tel quel.** L'app résume le texte écrit avant un appel d'outil, c'est pourquoi le chef écrit ses lignes d'état à la fin du tour.
- **Le worktree d'un ouvrier part de la branche par défaut du dépôt.** L'ouvrier s'aligne d'abord sur `integration_branch`, et seul ce qui y est **commité** lui parvient.
- **Les tickets vivent dans le dépôt principal.** Le chef les crée et y note les décisions. Chaque ouvrier reçoit le chemin du sien et le met à jour lui-même.
- **Les agents d'un plugin ignorent** les champs `permissionMode`, `hooks` et `mcpServers`.
- **Coût** : plusieurs ouvriers en parallèle consomment ta limite plus vite. Un ouvrier coupé par la limite ne reprend pas de lui-même : le chef le signale et propose de le reprendre, avec tout son historique.

La première semaine, mets `max_parallel: 1` dans `.ork/config.md`, le temps de roder le format des rapports, le protocole de blocage et le seuil « plan requis ».
