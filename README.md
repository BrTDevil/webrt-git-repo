# WebRT git-repo

`git init` + creare de repository nou pe GitHub (sub contul tău), dintr-o singură comandă, rulată direct în folderul proiectului. Setează și remote-ul `origin`, gata de `git add` / `git commit` / `git push` — fără să te mai duci pe github.com să creezi manual repo-ul și să dai copy-paste la comenzi.

Folosește tokenul deja salvat de git (prin `credential.helper`) — nu are nevoie de `gh` CLI și nu cere autentificare separată, dacă ai deja un `git push` funcțional către GitHub.

## Compilare

```bash
cd git-repo
go build -o git-repo .
```

## Utilizare

Rulează din directorul proiectului (cel care va deveni repo-ul). `-name` e **obligatoriu** — dacă lipsește, unealta nu face nimic (nu atinge git, nu contactează GitHub), doar afișează eroarea și lista de opțiuni. Numele repo-ului nu se ghicește niciodată din numele directorului, ca să nu creezi din greșeală un repo cu alt nume decât ai vrut:

```bash
cd calea/catre/proiectul-meu
git-repo -name proiectul-meu
```

Fără alte opțiuni: creează un repo **privat**, sub contul `BrTDevil`.

## Opțiuni

| Flag | Implicit | Descriere |
|---|---|---|
| `-name` | — | Numele repo-ului pe GitHub (**obligatoriu**) |
| `-owner` | `BrTDevil` | Cont/organizație GitHub sub care se creează repo-ul |
| `-desc` | — | Descrierea repo-ului |
| `-branch` | `main` | Numele branch-ului implicit (doar la `git init`, dacă directorul nu e deja un repo git) |
| `-public` | `false` | Creează repo-ul public (implicit e privat) |
| `-force` | `false` | Suprascrie remote-ul `origin` dacă indică în altă parte |
| `-now` | `false` | După setup: `git add -A`, commit și push, tot dintr-o comandă |
| `-full` | `false` | Alias pentru `-now` |
| `-m` | — | Mesaj de commit, folosit cu `-now`/`-full` (implicit: generat automat) |

## Ce face, exact

1. Ia tokenul GitHub din credențialele deja salvate de git (`git credential fill`).
2. Verifică dacă repo-ul `owner/name` există deja pe GitHub; dacă nu, îl creează (`POST /user/repos` sau `/orgs/<owner>/repos`).
3. Dacă directorul curent nu e deja un repo git, rulează `git init -b <branch>`.
4. Adaugă remote-ul `origin` către noul repo, dacă nu există deja (sau `set-url`, cu `-force`, dacă există și indică în altă parte).
5. Setează `push.autoSetupRemote true` local, ca primul `git push` să seteze singur upstream-ul, fără `-u origin <branch>`.
6. Cu `-now`/`-full`: continuă cu `git add -A`, `git commit -m "<mesaj>"` și `git push`. Dacă nu-i nimic de comis, se oprește curat fără eroare.

E idempotent și sigur de rulat repetat (util mai ales cu `-now`, ca să-l folosești zilnic): repo-ul existent pe GitHub e refolosit, `git init` e sărit dacă există deja `.git`, iar `origin` e refolosit dacă indică deja spre repo-ul corect — nu se suprascrie nimic fără `-force`.

## Exemple

```bash
# setup inițial: repo privat, sub BrTDevil
git-repo -name proiect-nou

# nume și descriere custom
git-repo -name proiect-nou -desc "Site pentru clientul X"

# repo public
git-repo -name proiect-nou -public

# alt cont/organizație
git-repo -name proiect-nou -owner alt-cont-github

# setup + add + commit + push, tot dintr-o comandă (mesaj auto-generat)
git-repo -name proiect-nou -now

# la fel, cu mesaj de commit custom
git-repo -name proiect-nou -now -m "adăugat formular de contact"
```

## Instalare globală (`webrt-git-repo` / `webrt:git-repo`, în orice terminal)

```bash
cd git-repo
go build -o git-repo .

sudo mkdir -p /opt/webrt-tools/bin
sudo install -m 755 git-repo /opt/webrt-tools/bin/git-repo

sudo ln -sf /opt/webrt-tools/bin/git-repo /usr/local/bin/webrt-git-repo
sudo ln -sf /opt/webrt-tools/bin/git-repo /usr/local/bin/webrt:git-repo
```

După asta, din orice director, în orice terminal:

```bash
cd proiectul-meu
webrt-git-repo
# sau
webrt:git-repo
```

Dezinstalare:

```bash
sudo rm -f /usr/local/bin/webrt-git-repo /usr/local/bin/webrt:git-repo /opt/webrt-tools/bin/git-repo
```
