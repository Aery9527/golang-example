#!/usr/bin/env bash
set -e
set -u
set -o pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

MODE="${1:?mode must be ci or dev}"
case "$MODE" in
    ci|dev)
        ;;
    *)
        echo "mode must be ci or dev: got '$MODE'" >&2
        exit 2
        ;;
esac
shift || true

GO_BIN="$(command -v go 2>/dev/null || command -v go.exe 2>/dev/null || true)"
if [[ -z "$GO_BIN" ]]; then
    echo "go executable not found in PATH" >&2
    exit 127
fi

TEST_ROOTS=("internal" "pkg")
ARTIFACT_DIR="$REPO_ROOT/test-output/${MODE}-test"
mkdir -p "$ARTIFACT_DIR"

TARGETS=()
for root in "${TEST_ROOTS[@]}"; do
    TARGETS+=("./${root}/...")
done

EXTRA_ARGS=("$@")
STDOUT_FILE="$ARTIFACT_DIR/stdout.log"
for arg in "${EXTRA_ARGS[@]}"; do
    if [[ "$arg" == "-json" ]]; then
        STDOUT_FILE="$ARTIFACT_DIR/stdout.jsonl"
        break
    fi
done

SIBLING_STDOUT_FILE="$ARTIFACT_DIR/stdout.log"
if [[ "$STDOUT_FILE" == "$SIBLING_STDOUT_FILE" ]]; then
    SIBLING_STDOUT_FILE="$ARTIFACT_DIR/stdout.jsonl"
fi

EXIT_CODE=0
cleanup() {
    printf '%s\n' "$EXIT_CODE" > "$ARTIFACT_DIR/exit-code.txt"
}
trap cleanup EXIT

rm -f "$SIBLING_STDOUT_FILE"
: > "$STDOUT_FILE"
: > "$ARTIFACT_DIR/stderr.log"

validate_extra_args() {
    local expecting_value=""
    local arg
    for arg in "$@"; do
        if [[ -n "$expecting_value" ]]; then
            case "$arg" in
                ./...|...|*/...)
                    echo "invalid extra arg '$arg': explicit package/path scope widening is not allowed; package patterns are fixed to ./internal/... and ./pkg/..." >&2
                    exit 2
                    ;;
            esac
            if [[ "$expecting_value" == "-coverprofile" ]]; then
                echo "invalid extra arg '$expecting_value': -coverprofile is reserved for the runner's dev-mode coverage artifact path" >&2
            elif [[ "$arg" == -* ]]; then
                echo "invalid extra arg '$arg': expected a value for '$expecting_value'; package patterns are fixed to ./internal/... and ./pkg/..." >&2
            fi
            if [[ "$expecting_value" == "-coverprofile" || "$arg" == -* ]]; then
                exit 2
            fi
            expecting_value=""
            continue
        fi

        if [[ "$arg" != -* ]]; then
            echo "invalid extra arg '$arg': package patterns are fixed to ./internal/... and ./pkg/...; extra args may only be flags or flag values" >&2
            exit 2
        fi

        case "$arg" in
            -tags)
                expecting_value="$arg"
                ;;
            -coverprofile)
                expecting_value="$arg"
                ;;
            -coverprofile=*)
                echo "invalid extra arg '$arg': -coverprofile is reserved for the runner's dev-mode coverage artifact path" >&2
                exit 2
                ;;
            -bench|-benchtime|-count|-covermode|-coverpkg|-cpu|-list|-outputdir|-parallel|-run|-shuffle|-skip|-timeout|-vet)
                expecting_value="$arg"
                ;;
        esac
    done

    if [[ -n "$expecting_value" ]]; then
        if [[ "$expecting_value" == "-coverprofile" ]]; then
            echo "invalid extra arg '$expecting_value': -coverprofile is reserved for the runner's dev-mode coverage artifact path" >&2
        else
            echo "invalid extra arg '$expecting_value': expected a value; package patterns are fixed to ./internal/... and ./pkg/..." >&2
        fi
        exit 2
    fi
}

GO_ARGS=("test")
if [[ "$MODE" == "ci" ]]; then
    GO_ARGS+=("-short")
fi
if [[ "$MODE" == "dev" ]]; then
    GO_ARGS+=("-coverprofile=$ARTIFACT_DIR/coverage.out")
fi
GO_ARGS+=("${EXTRA_ARGS[@]}" "${TARGETS[@]}")

{
    printf 'go'
    for arg in "${GO_ARGS[@]}"; do
        printf ' %q' "$arg"
    done
    printf '\n'
} > "$ARTIFACT_DIR/command.txt"

if ! VALIDATION_OUTPUT="$(validate_extra_args "${EXTRA_ARGS[@]}" 2>&1)"; then
    EXIT_CODE=2
    if [[ -n "$VALIDATION_OUTPUT" ]]; then
        printf '%s\n' "$VALIDATION_OUTPUT" | tee "$ARTIFACT_DIR/stderr.log" >&2
    fi
    exit "$EXIT_CODE"
fi

set +e
(
    cd "$REPO_ROOT"
    "$GO_BIN" "${GO_ARGS[@]}" >"$STDOUT_FILE" 2>"$ARTIFACT_DIR/stderr.log"
)
EXIT_CODE=$?
set -e

cat "$STDOUT_FILE"
if [[ -s "$ARTIFACT_DIR/stderr.log" ]]; then
    cat "$ARTIFACT_DIR/stderr.log" >&2
fi

if [[ "$MODE" == "dev" && -f "$ARTIFACT_DIR/coverage.out" ]]; then
    "$GO_BIN" tool cover -func="$ARTIFACT_DIR/coverage.out" | tee "$ARTIFACT_DIR/coverage-summary.txt"
fi

exit "$EXIT_CODE"
