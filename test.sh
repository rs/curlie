#!/bin/bash
# Compile, then run all ProgramBench test branches for rs__curlie.5dfcbb1.
#
# Test suites (one tarball per "branch") are downloaded from the public
# ProgramBench-Tests HuggingFace dataset on first run and cached under
# .programbench/tests/. Each branch is extracted into .programbench/run/<branch>,
# the freshly built ./executable is copied in (replacing the linux binary the
# tarball ships), and the branch's own eval/run.sh drives pytest, writing JUnit
# XML to eval/results.xml. Tests listed in .programbench/ignored_tests.txt are
# excluded from the final score, mirroring `programbench info`.
set -euo pipefail
cd "$(dirname "$0")"

# The reference results were produced in UTC containers; datetime-handling
# tests are timezone-sensitive, so pin TZ for reproducibility everywhere.
export TZ=UTC

# Containers resolve the temp dir to /tmp; macOS defaults to /var/folders/...,
# and some tests assert temp paths, so pin it to the container default.
export TMPDIR=/tmp

# Retry individually failed tests, as programbench eval does, to absorb
# flakes from tests that race on shared files under pytest-xdist.
export PYTEST_ADDOPTS="${PYTEST_ADDOPTS:-} --reruns=2 --reruns-delay=1"

# Bound test-phase memory: `-n auto` in a branch's run.sh would otherwise spawn
# one worker per core (14 here). Cap the auto worker count so parallel workers
# plus the binary under test stay well within the RAM budget. Override per
# project via .programbench/env if a suite needs more or fewer.
export PYTEST_XDIST_AUTO_NUM_WORKERS="${PYTEST_XDIST_AUTO_NUM_WORKERS:-4}"

# Bound compile-phase memory: `cargo build` and `go build` default to one
# compiler process per core (14), and each rustc can use 1-2GB, spiking well
# past the RAM budget. Cap build parallelism for both toolchains.
export CARGO_BUILD_JOBS="${CARGO_BUILD_JOBS:-2}"
export GOFLAGS="${GOFLAGS:-} -p=2"

# The benchmark containers shipped an older git that names the initial branch
# "master"; modern git names it "main". Tests that `git init` a repo and check
# refs/heads/master (or expect `checkout -b main` to be new) then diverge. Pin
# an isolated git config so every git in the run defaults to master, matching
# the containers, and mark test dirs safe so git commands do not refuse them.
export GIT_CONFIG_GLOBAL="$PWD/.programbench/gitconfig"
export GIT_CONFIG_SYSTEM=/dev/null
printf '[init]\n\tdefaultBranch = master\n[safe]\n\tdirectory = *\n' > "$GIT_CONFIG_GLOBAL"

# Optional per-project environment overrides (kept in the repo).
[ -f .programbench/env ] && . .programbench/env

INSTANCE=rs__curlie.5dfcbb1
BASE_URL="https://huggingface.co/datasets/programbench/ProgramBench-Tests/resolve/main/$INSTANCE/tests"
PB=.programbench

./compile.sh

[ -d "$PB/venv" ] || python3 -m venv "$PB/venv"
"$PB/venv/bin/pip" install -q pytest pytest-timeout pytest-xdist pytest-rerunfailures pytest-dependency tomli libtmux pexpect
export PATH="$PWD/$PB/venv/bin:$PATH"

# macOS has no GNU coreutils `timeout` and ships BSD `script` (incompatible
# flags: no -c, no long options). Many Linux-authored suites shell out to these,
# so provide translating shims on PATH; otherwise those tests die with
# "command not found" / "illegal option" instead of running. The shims call the
# real /usr/bin/script and never shadow a coreutils `timeout` if one is present.
SHIM="$PWD/$PB/shims"
mkdir -p "$SHIM"
if ! command -v timeout >/dev/null 2>&1; then
    cat > "$SHIM/timeout" <<'SH'
#!/bin/bash
# Minimal GNU-timeout shim: [-s SIG] [-k DUR] DURATION CMD...; returns 124 on timeout.
[ "$1" = "-s" ] && shift 2; [ "$1" = "-k" ] && shift 2
dur="$1"; shift
"$@" & pid=$!
( sleep "$dur" 2>/dev/null; kill -TERM "$pid" 2>/dev/null; sleep 1; kill -KILL "$pid" 2>/dev/null ) & w=$!
wait "$pid" 2>/dev/null; rc=$?
if kill -0 "$w" 2>/dev/null; then kill "$w" 2>/dev/null; else rc=124; fi
exit $rc
SH
    chmod +x "$SHIM/timeout"
fi
cat > "$SHIM/script" <<'SH'
#!/bin/bash
# Translate GNU `script [-a][-e][-f][-q][-c CMD] [FILE]` to BSD `script [-aeq] [FILE [cmd...]]`.
# BSD script echoes the child's closing EOF as a literal "^D\b\b" at the very
# start of the typescript; GNU script does not, and suites that assert exact
# output would see that 4-byte prefix. Strip it (harmless to length-only checks).
opts=""; cmd=""; file="/dev/null"
while [ $# -gt 0 ]; do
  case "$1" in
    -c) cmd="$2"; shift 2;;
    --command=*) cmd="${1#--command=}"; shift;;
    -*c) case "$1" in *q*) opts="$opts -q";; esac; case "$1" in *a*) opts="$opts -a";; esac
         case "$1" in *e*) opts="$opts -e";; esac; cmd="$2"; shift 2;;
    -*) case "$1" in *q*) opts="$opts -q";; esac; case "$1" in *a*) opts="$opts -a";; esac
        case "$1" in *e*) opts="$opts -e";; esac; shift;;
    *) file="$1"; shift;;
  esac
done
if [ -n "$cmd" ]; then
  /usr/bin/script $opts "$file" /bin/sh -c "$cmd" | perl -pe 's/\x5e\x44\x08\x08//g'
  exit "${PIPESTATUS[0]}"
else
  exec /usr/bin/script $opts "$file"
fi
SH
chmod +x "$SHIM/script"
export PATH="$SHIM:$PATH"

# Terminal-UI suites drive the binary through tmux (libtmux). Two macOS issues:
# (1) the default login shell is an interactive zsh whose nvm/conda/theme init
# takes ~1s and prints banners, so a command sent to a pane has not run yet when
# the test captures the pane; force panes to a bare, instant /bin/sh. (2) libtmux
# talks to the *default* tmux server, which may be the user's own; point tmux at
# a private socket dir so our sessions and kill-server never touch it. Harmless
# for suites that never launch tmux.
export SHELL=/bin/sh
export TMUX_TMPDIR="$PWD/$PB/tmux"
mkdir -p "$TMUX_TMPDIR"
tmux kill-server 2>/dev/null || true

mkdir -p "$PB/tests"
while read -r branch; do
    tar="$PB/tests/$branch.tar.gz"
    [ -f "$tar" ] || curl -fsSL -o "$tar" "$BASE_URL/$branch.tar.gz"
    # The extra "workspace" path segment matters: a few suites assert that
    # paths printed by the tool contain the substring "workspace".
    dir="$PB/run/$branch/workspace"
    rm -rf "$PB/run/$branch" && mkdir -p "$dir"
    # Seed the run dir with the project's tracked files, then overlay the test
    # tarball — in the real ProgramBench eval the tests run inside the
    # submission's own tree, and some suites read project files (testdata/,
    # docs, ...) that the tarball does not ship.
    git ls-files -z | tar --null -cf - -T - | tar -xpf - -C "$dir"
    tar xzf "$tar" -C "$dir"
    # ProgramBench's eval seeds a synthetic git repo in the workspace with one
    # commit of the whole tree (deterministic fixed identity/date). Tools that
    # locate the repo root via .git (rumdl, ripsecrets, ...) find this run dir,
    # and tools that read commit history (git-graph, ...) see a non-empty repo
    # instead of erroring on a commit-less one.
    git -C "$dir" init -q
    ( cd "$dir" && git add -A 2>/dev/null && \
        GIT_AUTHOR_DATE='2000-01-01T00:00:00Z' GIT_COMMITTER_DATE='2000-01-01T00:00:00Z' \
        git -c user.email=gold@local -c user.name=gold -c commit.gpgsign=false \
        commit -q --allow-empty -m gold ) >/dev/null 2>&1 || true
    cp executable "$dir/executable" && chmod +x "$dir/executable"
    # The suites were authored in containers whose workspace root is
    # /workspace, and some tests/goldens embed that absolute path. Rewrite it
    # to this run dir in every text file so path-sensitive assertions hold.
    # LC_ALL=C keeps BSD sed from aborting the whole xargs batch on a file
    # with non-UTF-8 bytes ("illegal byte sequence"), which would leave later
    # files (e.g. HTML goldens) un-rewritten.
    # Also use signal-based timeouts (as programbench eval does) so a
    # timing-out test fails cleanly instead of killing the pytest worker.
    grep -rlI --null -- /workspace "$dir" | LC_ALL=C xargs -0 sed -i.pbbak "s|/workspace|$PWD/$dir|g" || true
    find "$dir" -name '*.pbbak' -delete
    # Some run.sh scripts install the binary into /usr/local/bin so PATH-based
    # invocations (git hooks, ...) can find it; redirect that into a run-local
    # bin dir on PATH instead of requiring root.
    bindir="$PWD/$PB/run/$branch/bin"
    mkdir -p "$bindir"
    sed -i.pbbak -e 's/--timeout-method=thread/--timeout-method=signal/g' \
        -e "s|/usr/local/bin|$bindir|g" "$dir/eval/run.sh" \
        && rm -f "$dir/eval/run.sh.pbbak"
    echo "=== branch $branch ==="
    (cd "$dir" && PATH="$bindir:$PATH" bash eval/run.sh > run.log 2>&1 || true)
done < "$PB/branches.txt"

python3 - <<'EOF'
import xml.etree.ElementTree as ET
from pathlib import Path

def read_list(path: Path) -> set:
    if not path.exists():
        return set()
    return {l for l in map(str.strip, path.read_text().splitlines()) if l and not l.startswith("#")}


# ignored_tests.txt comes from ProgramBench (tests unreliable on the gold
# binary); local_ignored_tests.txt is course-side, for tests that are
# environment-sensitive outside the benchmark containers (each entry carries
# a comment explaining why).
ignored = read_list(Path(".programbench/ignored_tests.txt")) | read_list(
    Path(".programbench/local_ignored_tests.txt")
)
total = passed = dropped = 0
for branch in Path(".programbench/branches.txt").read_text().split():
    xml = Path(f".programbench/run/{branch}/workspace/eval/results.xml")
    if not xml.exists():
        print(f"{branch}: NO RESULTS (see .programbench/run/{branch}/workspace/run.log)")
        continue
    # pytest-rerunfailures records every attempt as its own <testcase> (the
    # non-final attempts are empty), so aggregate attempts per test name: a
    # test fails only if its final attempt carries a failure/error child.
    # Parametrized test IDs can embed workspace paths, which the harness
    # rewrote to the local run dir; map them back so names match the
    # benchmark's ignore lists.
    prefix = str(Path.cwd() / f".programbench/run/{branch}/workspace")
    cases: dict[str, set] = {}
    for case in ET.fromstring(xml.read_text()).iter("testcase"):
        name = f"{case.get('classname')}.{case.get('name')}".replace(prefix, "/workspace")
        cases.setdefault(name, set()).update(c.tag for c in case)
    n = ok = skip = 0
    for name, tags in cases.items():
        if f"{branch}/{name}" in ignored:
            skip += 1
            continue
        n += 1
        ok += not (tags & {"failure", "error", "skipped"})
    print(f"{branch}: {ok}/{n} passed ({skip} ignored)")
    total += n
    passed += ok
    dropped += skip
print(f"\nTOTAL: {passed}/{total} passed ({100 * passed / total:.1f}%), {dropped} tests ignored")
EOF
