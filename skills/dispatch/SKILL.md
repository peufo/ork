---
name: dispatch
description: Lancer les ouvriers ork et traiter leurs rapports. À charger par le chef avant de lancer un ticket ou dès qu'un rapport d'ouvrier arrive.
---

# Lancer les travaux

Si `.ork/tickets/.stop` existe, un arrêt propre est en cours : ne lance rien et suis `ork:stop`.

Quand des tickets sont `ready`, lance-les **sans redemander l'autorisation**, sauf si une règle de `## Ask before` couvre le ticket ou si le développeur a dit le contraire.

**Parallèle ou séquence**, c'est ta décision :
- ne lance jamais un ticket dont une dépendance (`depends_on`) n'est pas `done`, ni en PLAN MODE ni en EXECUTION MODE : l'ouvrier part de `integration_branch` et ne verrait pas le code dont il dépend. Le ticket reste `ready` et part dès que sa dépendance est intégrée ;
- ne lance jamais en parallèle deux tickets dont les `zones` se recoupent. Le second attend que le premier soit intégré ;
- respecte `max_parallel` (2 par défaut) ;
- en cas de doute, séquence. Un conflit de fusion coûte plus cher qu'une attente.

**Plan ou exécution directe** :
- si `plan_required: true`, ou si le ticket correspond aux règles `## Plan required when`, lance l'ouvrier en **PLAN MODE** et passe le ticket en `planning`. L'ouvrier écrit son plan dans `## Plan` et passe le ticket en `plan-review`. Tu présentes ensuite le plan au développeur, en **cinq lignes maximum** ;
- après validation, relance **le même ouvrier** (via `SendMessage` à son identifiant) avec « Plan validé, passe en EXECUTION MODE », en ajoutant les corrections éventuelles ;
- sinon, lance directement en **EXECUTION MODE**.

**Modèle et effort** : choisis-les au lancement, passe-les dans les paramètres `model` et `effort` de l'appel, et note-les dans le ticket. Le développeur peut les corriger avant le lancement : sa valeur prime.

| Cas | Modèle | Effort |
|---|---|---|
| PLAN MODE | `opus` | `high` |
| EXECUTION MODE sans plan, petit ticket sans doute (1 à 3 fichiers, critères nets, aucune règle « Plan required when ») | `sonnet` | `medium` |
| EXECUTION MODE sans plan, tous les autres cas | `opus` | défaut (n'envoie pas `effort`) |
| Relance après un `FAILED` | un cran au-dessus (`sonnet` → `opus`) | `high` |
| INTEGRATION MODE avec un nouvel ouvrier | `opus` | défaut |

En cas de doute entre deux lignes, prends la plus haute. Un ouvrier relancé par `SendMessage` garde le modèle et l'effort de son lancement. Si `.ork/config.md` donne d'autres règles, elles priment.

**Le message de délégation** à `ork:worker` contient toujours :
1. le mode (`PLAN MODE` ou `EXECUTION MODE`) ;
2. `TICKET FILE: <chemin absolu>`, le fichier du ticket dans le dépôt principal. L'ouvrier le lit et le met à jour lui-même ;
3. la `integration_branch`, la commande de tests et les notes utiles de `.ork/config.md` ;
4. en EXECUTION MODE, la ligne `AUTO REVIEW: <commands>` si `auto_review` est configuré.

Fais commencer la `description` du lancement par l'identifiant du ticket (`t38 plan identité visuelle`) : la micro-app s'en sert pour afficher l'agent sur sa carte. Note l'identifiant de l'agent dans le ticket (`agent:`) dès que tu l'as, et passe le statut à `planning` ou `in-progress`.

# Quand un ouvrier rend compte

Le rapport est court : l'ouvrier a déjà mis son ticket à jour. Relis le ticket avant de décider. S'il est resté à jour, ne le réécris pas. Si le rapport contient `TICKET NOT UPDATED`, l'écriture lui a été refusée : reporte toi-même dans le ticket ce que dit le rapport, et ajoute `[WARN]` à ta réponse.

Pour chaque rapport :
- `STATUS: PLAN READY` : **urgent** ;
- `STATUS: BLOCKED` : **urgent**. Exception : si la `confidence` de la recommandation atteint `confidence_threshold`, note la réponse dans `## Questions` avec `(auto N/10)`, repasse le statut à `in-progress`, relance l'ouvrier par `SendMessage` avec cette réponse, et ajoute une ligne `` [AUTO] `t14-export-pdf` : garder l'export CSV (9/10) `` à ta prochaine réponse ;
- `STATUS: DONE` : coche d'office dans `## Review` les points qui atteignent `confidence_threshold`, puis décide de la relecture (ci-dessous) ;
- `STATUS: FAILED` : **urgent**, avec ta proposition (relancer un cran au-dessus, découper, abandonner). Si une règle de `## No need to ask` couvre ta proposition, applique-la et signale-la ;
- `STATUS: PAUSED` : l'ouvrier s'est arrêté à ta demande, et le ticket garde son statut. Reprendre, c'est relancer **le même ouvrier** par `SendMessage` (« reprends »), ou, s'il n'est plus joignable, suivre `ork:integrate` ;
- **fin sans rapport sur une erreur d'API** (limite d'usage, surcharge) : la notification d'échec nomme l'erreur et ne contient pas de ligne `STATUS:`. Ce n'est pas un `FAILED` : l'ouvrier a été coupé. Note-le dans le journal, laisse le ticket dans son statut, et ajoute `` [STOP] `t13-slug` : limite d'usage, « reprends t13 » après le reset ``. Reprendre, c'est relancer **le même ouvrier** par `SendMessage` (« reprends là où tu t'es arrêté »), sans cran au-dessus. Ne lance rien de nouveau avant le reset de la limite.

Quand le développeur répond à une question, mets le ticket à jour, puis relance l'ouvrier concerné via `SendMessage`. S'il n'est plus joignable, charge `ork:integrate`.

Puis vérifie si un ticket `ready` en attente peut maintenant partir.

**Relecture ou intégration directe.** Après un `DONE`, intègre **sans attendre le développeur** (charge `ork:integrate`) si toutes ces conditions sont réunies :
- `review: per-ticket` dans `.ork/config.md`, et pas de `review: required` dans le ticket ;
- aucune règle de `## Review required when` ne s'applique, d'après le ticket et les fichiers touchés (`git diff --name-only <integration_branch>...<branch>`). Si une règle s'applique, pose `review: required` ;
- aucune règle de `## Ask before` ne couvre l'intégration de ce ticket ;
- le ticket n'est pas passé par un plan ;
- toutes les commandes de `AUTO REVIEW` ont tourné, et `## Review` n'en note aucune `unavailable` ;
- aucun point de `## Review` sous `confidence_threshold` (sans seuil : aucun point). Ceux qui l'atteignent sont cochés d'office ;
- les tests sont verts après le rebase.

L'intégration réussie donne `` [OK] `t12-slug` -> `main` : sans relecture `` dans ta prochaine réponse. Si une condition manque, ou si l'intégration s'arrête en route, le ticket reste `in-review`, avec `` [NOTE] `t12-slug` : <ce qui demande la relecture> ``.

**`confidence_threshold`** : ce qui atteint le seuil passe sans le développeur. Un point sans note n'est jamais coché d'office. Absent : tout est montré. `0` : tout passe. Le seuil ne s'applique jamais à un ticket `review: required`, à une décision couverte par `## Ask before`, ni aux règles hors dosage.

$ARGUMENTS
