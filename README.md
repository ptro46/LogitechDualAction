# Logitech Dual Action — console Go pour macOS

Affichage en direct des boutons B1 à B12, de la croix directionnelle, et de tous les axes HID exposés par la manette. Repérage conforme à `LogitechDualAction.png`.

## Lancer

```sh
cd /Users/ptro/Projets/LogitechDualAction
./logitech-dual-action
```

Brancher la manette avant ou après le démarrage. L’écran se met à jour sur place ; Ctrl+C termine le programme. En cas de déconnexion, il attend automatiquement son retour. Si plusieurs Dual Action sont présentes, la première retournée par macOS est affichée.

```sh
./logitech-dual-action -list            # manettes HID détectées
./logitech-dual-action -once            # un état instantané
./logitech-dual-action -plain           # journal des changements
./logitech-dual-action -interval 100ms   # fréquence de lecture
./logitech-dual-action -vendor 0x046d -product 0xc216
```

Sans terminal interactif, le programme journalise automatiquement les changements. Chaque lecture décrit l'état actuel : un appui plus bref que l'intervalle de lecture peut être manqué.

## Commandes

| Bouton | Emplacement |
|---|---|
| B1 / B2 / B3 / B4 | Face gauche / bas / droite / haut |
| B5 / B6 | L1 / R1 |
| B7 / B8 | L2 / R2 |
| B9 / B10 | Select / Start |
| B11 / B12 | Clic stick gauche / droit |

Les numéros proviennent des usages HID des boutons. Les axes sont affichés avec leur nom HID, leur valeur brute et une normalisation de -1 à +1. X/Y correspondent généralement au stick gauche, Z/Rz au stick droit. La croix affiche les huit directions ou « repos ».

Le bouton MODE peut modifier en interne le rôle de la croix et du stick gauche sans transmettre un bouton distinct à l’ordinateur. Le programme affiche tous les boutons HID exposés, y compris ceux au-delà de B12, mais ne peut pas lire un état que le matériel ne transmet pas. Le repérage physique des axes est donc à vérifier selon le mode choisi.

## Compiler et vérifier

Go 1.25 ou supérieur et les outils de développement Apple sont nécessaires (`xcode-select --install` si absents). Le programme utilise un petit pont C vers les frameworks système IOKit et CoreFoundation ; CGO doit être activé. La version console utilise les bibliothèques standard de Go et ce pont C. La version graphique ajoute Ebitengine et ses dépendances Go, téléchargées automatiquement à la première compilation et verrouillées dans `go.mod` / `go.sum`.

```sh
go build -o logitech-dual-action .
go test ./...
go vet ./...
```

Cette version cible macOS. Le code de présentation est en Go ; la lecture USB HID est spécifique à macOS. Les autres systèmes retournent un message explicite.

Référence API : [IOHIDDeviceGetValue, documentation Apple](https://developer.apple.com/documentation/iokit/1588657-iohiddevicegetvalue).

## Vérification effectuée

Compilation macOS x86_64, tests Go (directions de la croix, normalisation, distinction des états indisponibles) et `go vet` réussis. Le registre USB du Mac identifie une Logitech Dual Action `046d:c216`. La lecture réelle depuis l'environnement isolé de Codex a retourné `kIOReturnNotPermitted` : les appuis physiques et la reconnexion restent à vérifier depuis Terminal.

En cas de refus HID, lancer l'exécutable depuis Terminal. Si le refus persiste, vérifier les autorisations de Surveillance de l’entrée du terminal dans Réglages Système > Confidentialité et sécurité.

## Version graphique

La console conserve son exécutable et son fonctionnement. Une seconde version native macOS dessine la manette : B1–B12 passent au vert à l'appui, les diagonales de la croix allument deux directions et les sticks se déplacent selon les axes X/Y et Z/Rz. Les clics des sticks allument B11/B12. Une commande non lisible reste grisée ; les positions reviennent au neutre si la manette est déconnectée. La fenêtre est redimensionnable et conserve les proportions du dessin. Échap et Cmd+Q permettent aussi de quitter.

Double-cliquer sur `Logitech Dual Action.app`, ou lancer depuis Terminal :

```sh
cd /Users/ptro/Projets/LogitechDualAction
./logitech-dual-action-graphique
```

Fermer la fenêtre ou utiliser Cmd+Q pour quitter. Le statut dans la fenêtre indique la connexion, l’absence de manette ou un refus d’accès HID. Le mode MODE reste une indication fixe car son état n’est pas transmis comme un bouton indépendant.

```sh
./logitech-dual-action-graphique -demo  # animation avec des valeurs simulées
./logitech-dual-action-graphique -demo -preview /tmp/dual-action.png
```

Le mode démo est explicitement marqué à l’écran et ne lit pas la manette. L’option `-preview` exporte une image puis quitte.

Pour recompiler :

```sh
make console   # version console
make app       # version graphique + application macOS signée localement
make test      # tests et analyse statique des deux variantes
```

Sans Make : `go build -tags gui -o logitech-dual-action-graphique .`. Le tag `gui` sélectionne le point d’entrée graphique ; sans ce tag, la compilation reste celle de la console. Le code Go lit et convertit les états HID. La fenêtre, le dessin vectoriel et les animations sont écrits en Go dans `graphic_darwin.go`, avec Ebitengine 2.10.5. Le fichier `graphic_state.go` convertit les états USB vers les commandes graphiques. Les anciens fichiers `graphic_darwin.m` et `graphic_darwin.h` ont été supprimés. Seul le pont de lecture HID (`hid_darwin.c` / `.h`) reste en C. Les bibliothèques graphiques peuvent elles-mêmes appeler les API natives de macOS en interne : le code applicatif graphique est en Go, sans fichier Objective-C à maintenir dans ce projet.

Compilation macOS x86_64, tests et `go vet` réussis pour les deux variantes. Les tests graphiques vérifient le mapping B1/B4/B12, le relâchement, les quatre axes, la croix et la remise au neutre après déconnexion. Dans l’environnement isolé de Codex, le démarrage de la fenêtre reste bloqué dans l’initialisation native d’Ebitengine, avant le dessin (erreurs de connexion aux services macOS). Aucun nouvel aperçu graphique n’a donc été validé pour cette version Go. Le lancement de la fenêtre, le rendu et les appuis physiques restent à vérifier depuis la session normale du Mac. L’application est signée localement, sans notarisation Apple.

Références : [Ebitengine](https://ebitengine.org/) et [dessin vectoriel](https://ebitengine.org/en/examples/vector.html).
