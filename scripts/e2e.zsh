#!/usr/bin/env zsh
set -euo pipefail

LYRA_BIN=${LYRA_BIN:-${0:A:h:h}/bin/lyra}
WORK=${WORK:-$(mktemp -d)}
PORT=18080
OTHER_PORT=18081
LOGIN=tester@example.com
export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1 GIT_TERMINAL_PROMPT=0
export LYRA_URL=http://127.0.0.1:$PORT

step() { print -P "\n%F{cyan}== $*%f" }
expect_status() {
  local want=$1; shift
  local got
  got=$(curl -s -o /dev/null -w '%{http_code}' "$@")
  if [[ $got != $want ]]; then print "FAIL: want $want got $got for $*"; exit 1; fi
  print "ok: HTTP $got"
}

print "work dir: $WORK"
mkdir -p $WORK/keys
ssh-keygen -q -t ed25519 -N '' -C 'agent-test-key' -f $WORK/keys/agent
ssh-keygen -q -t ed25519 -N '' -C 'stranger-key' -f $WORK/keys/stranger

for port in $PORT $OTHER_PORT; do
  if curl -s -o /dev/null http://127.0.0.1:$port/healthz; then print "FAIL: port $port already in use (stale server?)"; exit 1; fi
done
$LYRA_BIN serve --listen 127.0.0.1:$PORT --data $WORK/data --trusted-proxy 127.0.0.1 --allow-login $LOGIN > $WORK/server.log 2>&1 &
SERVER=$!
$LYRA_BIN serve --listen 127.0.0.1:$OTHER_PORT --data $WORK/data-other --trusted-proxy 10.9.9.9 --allow-login $LOGIN > $WORK/server-other.log 2>&1 &
OTHER=$!
trap 'kill $SERVER $OTHER 2>/dev/null || true' EXIT
for _ in {1..50}; do curl -sf $LYRA_URL/healthz >/dev/null && curl -sf http://127.0.0.1:$OTHER_PORT/healthz >/dev/null && break; sleep 0.1; done

step "auth: no credentials -> 401"
expect_status 401 $LYRA_URL/api/me
step "auth: Tailscale header from trusted proxy"
curl -sf -H "Tailscale-User-Login: $LOGIN" -H 'Lyra-Agent: curl' $LYRA_URL/api/me
step "auth: Tailscale header with a login not allowed -> 403"
expect_status 403 -H 'Tailscale-User-Login: intruder@example.com' $LYRA_URL/api/me
step "auth: Tailscale header from an untrusted address -> 401"
expect_status 401 -H "Tailscale-User-Login: $LOGIN" http://127.0.0.1:$OTHER_PORT/api/me

step "bootstrap admin token through the Tailscale header (secret kept in $WORK/admin.token)"
curl -sf -H "Tailscale-User-Login: $LOGIN" -H 'Content-Type: application/json' \
  -d '{"name":"admin","scope":"admin"}' $LYRA_URL/api/tokens > $WORK/admin.json
jq -r .secret $WORK/admin.json >| $WORK/admin.token
jq 'del(.secret)' $WORK/admin.json
export LYRA_TOKEN=$(<$WORK/admin.token)
export LYRA_AGENT=e2e-agent LYRA_SESSION=session-123

step "cli: me"
$LYRA_BIN me
step "cli: repo create demo, other"
$LYRA_BIN repo create demo --description 'Demo repo'
$LYRA_BIN repo create other >/dev/null
step "cli: repo create with an invalid name -> error JSON on stderr, exit 1"
if $LYRA_BIN repo create 'bad/name' 2>$WORK/err.json; then print FAIL; exit 1; fi
cat $WORK/err.json
step "cli: repo list"
$LYRA_BIN repo list | jq -c '.[] | {name, default_branch, clone_url}'
step "cli: --jq filter (strings raw, other values as JSON)"
$LYRA_BIN repo list --jq '.[].name'
$LYRA_BIN repo get demo --jq '{name, branches}'
step "cli: missing argument -> usage error, exit 2"
set +e; $LYRA_BIN repo get 2>$WORK/err.json; code=$?; set -e
cat $WORK/err.json; [[ $code == 2 ]] && print "ok: exit $code"

step "git: push signed, unsigned and unknown-key commits plus a signed tag"
git init -q -b main $WORK/src
cd $WORK/src
git config user.name 'Test Agent'
git config user.email agent@example.com
git config gpg.format ssh
echo '# Demo' > README.md
mkdir -p docs && echo 'Hello' > docs/intro.md
git add . && git -c user.signingkey=$WORK/keys/agent commit -q -S -m 'Signed by the agent key'
echo 'unsigned' >> README.md && git commit -qam 'Unsigned commit'
echo 'stranger' >> README.md && git -c user.signingkey=$WORK/keys/stranger commit -qS -am 'Signed by an unregistered key'
git -c user.signingkey=$WORK/keys/agent tag -s v0.1.0 -m 'First tag'
git tag -a v0.0.1 -m 'Unsigned annotated tag' HEAD~2
git tag lightweight HEAD~1
git checkout -q -b side
echo 'side' > side.txt && git add side.txt && git commit -qm 'Side branch commit'
git -c user.signingkey=$WORK/keys/agent tag -s side-v1 -m 'Signed side tag'
git checkout -q main
GIT_MERGE_AUTOEDIT=no git -c user.signingkey=$WORK/keys/agent merge -q --no-ff -S -m 'Signed merge of a signed tag' side-v1
git cat-file commit HEAD | grep -q '^mergetag ' && print "ok: merge commit carries a mergetag header"
AUTH_HEADER="Authorization: Bearer $LYRA_TOKEN"
git -c http.extraHeader="$AUTH_HEADER" -c http.extraHeader='Lyra-Agent: e2e-agent' -c http.extraHeader='Lyra-Session: session-123' \
  push -q http://127.0.0.1:$PORT/demo.git main --tags

step "git: large push with a tiny postBuffer (chunked request body)"
head -c 3000000 /dev/urandom > blob.bin
git add blob.bin && git commit -qm 'Add random blob'
git -c http.extraHeader="$AUTH_HEADER" -c http.postBuffer=1024 push -q http://127.0.0.1:$PORT/demo.git main
print "ok: pushed $(git rev-parse --short HEAD)"

step "git: clone with Basic auth through a credential helper (repo name without .git)"
git -c credential.helper= -c credential.helper="!f() { echo username=lyra; echo password=\$(cat $WORK/admin.token); }; f" \
  clone -q http://127.0.0.1:$PORT/demo $WORK/clone
diff -r --exclude=.git $WORK/src $WORK/clone && print "ok: clone matches source at $(git -C $WORK/clone rev-parse --short HEAD)"

step "git: clone and push through the Tailscale header from the trusted proxy, no token"
git -c http.extraHeader="Tailscale-User-Login: $LOGIN" clone -q http://127.0.0.1:$PORT/demo.git $WORK/clone-tailnet
git -C $WORK/clone-tailnet -c user.name=T -c user.email=t@example.com commit -q --allow-empty -m 'Pushed over the tailnet header'
git -C $WORK/clone-tailnet -c http.extraHeader="Tailscale-User-Login: $LOGIN" -c http.extraHeader='Lyra-Agent: tailnet-agent' push -q origin main
print "ok: tailnet push $(git -C $WORK/clone-tailnet rev-parse --short HEAD)"

step "signatures before registering the key"
$LYRA_BIN commit list demo | jq -c '.[] | {subject, status: .signature.status}'

step "cli: key add --signing --agent"
$LYRA_BIN key add --signing --agent claude --title 'e2e agent key' $WORK/keys/agent.pub
step "cli: key add rejects a private key"
if $LYRA_BIN key add --signing $WORK/keys/agent 2>$WORK/err.json; then print FAIL; exit 1; fi
cat $WORK/err.json
step "signatures after registering the key (commit list)"
$LYRA_BIN commit list demo | jq -c '.[] | {sha: .sha[0:7], subject, signature}'
step "tags"
$LYRA_BIN tag list demo | jq -c '.[] | {name, annotated, signature}'

step "forged commits signed by the registered key must be unverified"
cd $WORK/src
SIGNED=$(git log --format=%H --grep='^Signed by the agent key$' main)
TAMPERED=$(git cat-file commit $SIGNED | sed 's/^Signed by the agent key$/Tampered message/' | git hash-object -t commit -w --stdin)
UNSIGNED=$(git log --format=%H --grep='^Unsigned commit$' main)
git cat-file commit $UNSIGNED >| $WORK/payload.txt
ssh-keygen -q -Y sign -n file -f $WORK/keys/agent $WORK/payload.txt
WRONG_NS=$(awk -v sigfile=$WORK/payload.txt.sig '
  !done && /^$/ { first = 1; while ((getline line < sigfile) > 0) { printf "%s%s\n", (first ? "gpgsig " : " "), line; first = 0 } done = 1 }
  { print }' $WORK/payload.txt | git hash-object -t commit -w --stdin)
git -c http.extraHeader="$AUTH_HEADER" push -q http://127.0.0.1:$PORT/demo.git "${TAMPERED}:refs/heads/tampered" "${WRONG_NS}:refs/heads/wrong-namespace"
for sha in $TAMPERED $WRONG_NS; do
  forged=$($LYRA_BIN commit show demo $sha --jq .signature.status)
  [[ $forged == unverified ]] || { print "FAIL: forged $sha is $forged"; exit 1 }
  print "ok: ${sha[1,7]} $forged ($(git log -1 --format=%s $sha))"
done

step "cross-check every commit and tag with git verify-commit / verify-tag (independent allowed_signers)"
BARE=$WORK/data/repos/demo.git
print "$LOGIN namespaces=\"git\" $(cut -d' ' -f1,2 $WORK/keys/agent.pub)" >| $WORK/allowed_signers
for sha in $(git -C $BARE rev-list --all); do
  lyra=$($LYRA_BIN commit show demo $sha --jq .signature.status)
  if git -C $BARE -c gpg.ssh.allowedSignersFile=$WORK/allowed_signers verify-commit $sha 2>/dev/null; then expected=verified
  elif git -C $BARE cat-file commit $sha | grep -q '^gpgsig'; then expected=unverified
  else expected=unsigned; fi
  [[ $lyra == $expected ]] || { print "FAIL: commit $sha lyra=$lyra git=$expected"; exit 1 }
  print "ok: ${sha[1,7]} $lyra  $(git -C $BARE log -1 --format=%s $sha)"
done
for name in $(git -C $BARE for-each-ref --format='%(refname:short)' refs/tags); do
  lyra=$($LYRA_BIN tag list demo --jq ".[] | select(.name == \"$name\") | .signature.status")
  if git -C $BARE -c gpg.ssh.allowedSignersFile=$WORK/allowed_signers verify-tag $name 2>/dev/null; then expected=verified
  elif [[ $(git -C $BARE cat-file -t $name) == tag ]] && git -C $BARE cat-file tag $name | grep -q '^-----BEGIN SSH SIGNATURE-----'; then expected=unverified
  else expected=unsigned; fi
  [[ $lyra == $expected ]] || { print "FAIL: tag $name lyra=$lyra git=$expected"; exit 1 }
  print "ok: tag $name $lyra"
done

step "deleting the key turns its signatures unverified"
KEY_ID=$($LYRA_BIN key list --jq '.[0].id')
$LYRA_BIN key delete $KEY_ID >/dev/null
print "statuses without the key: $($LYRA_BIN commit list demo --jq '[.[].signature.status] | unique | join(",")')"
[[ $($LYRA_BIN commit list demo --jq '[.[] | select(.signature.status == "verified")] | length') == 0 ]] || { print "FAIL: still verified"; exit 1 }
$LYRA_BIN key add --signing --agent claude $WORK/keys/agent.pub >/dev/null
print "ok: key re-added; verified again: $($LYRA_BIN commit list demo --jq '[.[] | select(.signature.status == "verified")] | length') commits"

step "tokens: read-only token scoped to demo"
$LYRA_BIN token create --name ci-read --scope read --repo demo --expires 1h > $WORK/read.json
jq -r .secret $WORK/read.json >| $WORK/read.token
jq -c 'del(.secret)' $WORK/read.json
READ_HEADER="Authorization: Bearer $(<$WORK/read.token)"
LYRA_TOKEN=$(<$WORK/read.token) $LYRA_BIN repo list | jq -c '[.[].name]'
expect_status 403 -H "$READ_HEADER" $LYRA_URL/api/repos/other
git -c http.extraHeader="$READ_HEADER" clone -q http://127.0.0.1:$PORT/demo.git $WORK/clone-read && print "ok: read token can clone"
cd $WORK/clone-read && git commit -q --allow-empty -m 'should be rejected'
if git -c http.extraHeader="$READ_HEADER" push -q origin main 2>$WORK/push-err.txt; then print "FAIL: read token pushed"; exit 1; fi
print "ok: read token push rejected: $(grep -o 'HTTP [0-9]*\|403[^\n]*' $WORK/push-err.txt | head -1)"
if LYRA_TOKEN=$(<$WORK/read.token) $LYRA_BIN repo create sneaky 2>$WORK/err.json; then print FAIL; exit 1; fi
cat $WORK/err.json
$LYRA_BIN token list | jq -c '.[] | {id, name, scope, repos, prefix, expires: (.expires_at != null), used: (.last_used_at != null)}'
READ_ID=$(jq .token.id $WORK/read.json)
$LYRA_BIN token revoke $READ_ID
expect_status 401 -H "$READ_HEADER" $LYRA_URL/api/me

step "events record user, via, agent and session"
$LYRA_BIN event list | jq -c '.[] | {action, repo, actor, data}'

step "cli: repo delete needs --yes"
if $LYRA_BIN repo delete other 2>$WORK/err.json; then print FAIL; exit 1; fi
cat $WORK/err.json
$LYRA_BIN repo delete other --yes
[[ ! -d $WORK/data/repos/other.git ]] && print "ok: other.git removed"

print -P "\n%F{green}all checks passed%f (work dir $WORK)"
