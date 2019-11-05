# pagesjaunes

Le Domain Name System (DNS) sert d'annuaire pour internet, permettant de trouver l'addresse IP correspondant à un nom de domaine. Cette brique indispensable à tout système d'information tourne en general sur plusieurs machines, et ce afin d'assurer une continuité de service en cas de crash. Une synchronisation permanente des base de données de chaque serveur DNS est donc indispensable pour créer un service répliqué.

Ce projet vous invite à découvrir la puissance du protocal Raft en créant votre DNS distribué. Le protocal Raft vous permet de maintenir chaque membre à jour des derniers changements. Si une nouvelle donnée est enregistrée sur un noeud A, ce noeud envoie aux noeuds B et C cette donnée afin de maintenir le cluster synchronisé.

Ce DNS aura les caracteristiques suivantes:
- Le cluster est composé de 3 noeuds.
  Vous devrez executer 3 instances du même binaire sur des ports UDP differents pour créer ce cluster sur votre poste.

- Chaque serveur doit etre lancé avec son propre fichier de zone contenant les entrées DNS.
  Exemple: Le serveur "00" sera lancé avec son fichier de zone "z00.txt". Des fichiers d'exemple seront fournis.

- Les noeuds devront repliquer leurs bases de données via le protocol Raft.
  Ainsi, chaque membre du cluster contiendra toutes les entrées DNS de tout le monde.

- Le signal SIGHUP recharge "à chaud" le fichier de zone, nous permettant d'ajouter des entrée DNS via ce fichier.

Vous êtes libre d'utiliser le langage de votre choix, ainsi que toute librairie vous permettant de gagner du temps.
Ne ré-inventez pas la roue.

## Design

```
  ┌───────────┐             ┌───────────┐             ┌───────────┐
  │z01.txt    │             │z00.txt    │             │z02.txt    │
  │           │             │           │             │           │
  │bar.com    │             │foo.com    │             │baz.com    │
  └─────▲─────┘             └─────▲─────┘             └─────▲─────┘
        │                         │                         │      
    read│                     read│                     read│      
        │                         │                         │      
  ┌─────┴─────┐             ┌─────┴─────┐             ┌─────┴─────┐
  │           │  leader     │           │     leader  │           │
  │           │forwarding   │           │   forwarding│           │
  │ node 01   ├─────────────▶ node 00   ◀─────────────┤ node 02   │
  │           │             │           │             │           │
  │ follower  │             │ leader    │             │ follower  │
  │           │  replication│           │replication  │           │
  │           ◀─────────────┤           ├─────────────▶           │
  └───────────┘             └───────────┘             └───────────┘
```

## Usage

Lancer les 3 noeuds:
```
$ bin/dns-raft -id id0 -raft.addr ":8300" -dns.addr ":8600" -zone.file "./zones/z00.txt"
$ bin/dns-raft -id id1 -raft.addr ":8301" -dns.addr ":8601" -zone.file "./zones/z01.txt" -raft.join ":8300"
$ bin/dns-raft -id id2 -raft.addr ":8302" -dns.addr ":8602" -zone.file "./zones/z02.txt" -raft.join ":8300"
```

## DNS

Chaque noeud lis un [fichier de zone](zones/) à l'execution, et réplique ses données aux autres noeuds.

Resoudre les domaines depuis le premier noeud:
```
$ dig @127.0.0.1 -p 8600 foo.com
$ dig @127.0.0.1 -p 8600 bar.com
$ dig @127.0.0.1 -p 8600 baz.com
```

Resoudre les domaines depuis le second noeud:
```
$ dig @127.0.0.1 -p 8601 foo.com
$ dig @127.0.0.1 -p 8601 bar.com
$ dig @127.0.0.1 -p 8601 baz.com
```

Resoudre les domaines depuis le troisième noeud:
```
$ dig @127.0.0.1 -p 8602 foo.com
$ dig @127.0.0.1 -p 8602 bar.com
$ dig @127.0.0.1 -p 8602 baz.com
```

Ajouter une entrée DNS au fichier de zone "z00.txt":
```
$ echo 'database                 60 A     1.2.3.7' >> zones/z00.txt
```

Recharger le fichier de zone en mémoire:
```
$ pkill -SIGHUP dns-raft
```

Résoudre ce nouveau domaine depuis n'importe quel noeud:
```
$ dig @127.0.0.1 -p 8602 database.foo.com
```

Couper le 2ème noeud avec CTRL-C, puis ajouter une entrée sur "z00.txt"
```
$ echo 'bonjour                  60 A     1.2.3.40' >> zones/z02.txt
$ pkill -SIGHUP dns-raft
```

Relancer le 2ème noeud:
```
$ bin/dns-raft -id id1 -raft.addr ":8301" -dns.addr ":8601" -zone.file "./zones/z01.txt" -raft.join ":8300"
```

Est-ce que ce noeud à bien reçu les nouvelles données ?
```
$ dig @127.0.0.1 -p 8601 bonjour.baz.com
```
