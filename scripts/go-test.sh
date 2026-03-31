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

EXIT_CODE=0
cleanup() {
    printf '%s\n' "$EXIT_CODE" > "$ARTIFACT_DIR/exit-code.txt"
}
trap cleanup EXIT

find "$ARTIFACT_DIR" -mindepth 1 -maxdepth 1 -exec rm -rf {} +

TARGETS=()
for root in "${TEST_ROOTS[@]}"; do
    TARGETS+=("./${root}/...")
done

EXTRA_ARGS=("$@")

stdout_file_for_args() {
    local arg
    for arg in "$@"; do
        case "$arg" in
            -json|-json=true)
                printf '%s\n' "$ARTIFACT_DIR/stdout.jsonl"
                return 0
                ;;
        esac
    done

    printf '%s\n' "$ARTIFACT_DIR/stdout.log"
}

STDOUT_FILE="$(stdout_file_for_args "${EXTRA_ARGS[@]}")"
: > "$STDOUT_FILE"
: > "$ARTIFACT_DIR/stderr.log"

native_path() {
    local path="$1"

    if command -v wslpath >/dev/null 2>&1; then
        wslpath -w "$path"
        return 0
    fi

    if command -v cygpath >/dev/null 2>&1; then
        cygpath -w "$path"
        return 0
    fi

    printf '%s\n' "$path"
}

validate_coverpkg_value() {
    local value="$1"
    local part
    local found_scope=0
    local COVERPKG_PARTS=()

    IFS=',' read -r -a COVERPKG_PARTS <<< "$value"
    for part in "${COVERPKG_PARTS[@]}"; do
        case "$part" in
            ./internal/...|./pkg/...)
                found_scope=1
                ;;
            *)
                echo "invalid extra arg '$value': -coverpkg must stay within ./internal/... and ./pkg/... scopes" >&2
                return 1
                ;;
        esac
    done

    if [[ "$found_scope" -ne 1 ]]; then
        echo "invalid extra arg '$value': -coverpkg must stay within ./internal/... and ./pkg/... scopes" >&2
        return 1
    fi
}

validate_extra_args() {
    local expecting_value=""
    local arg
    for arg in "$@"; do
        if [[ -n "$expecting_value" ]]; then
            if [[ "$expecting_value" == "-coverprofile" ]]; then
                echo "invalid extra arg '$expecting_value': -coverprofile is reserved for the runner's dev-mode coverage artifact path" >&2
                return 1
            fi

            if [[ "$expecting_value" == "-coverpkg" ]]; then
                validate_coverpkg_value "$arg" || return 1
            fi

            expecting_value=""
            continue
        fi

        if [[ "$arg" != -* ]]; then
            echo "invalid extra arg '$arg': package patterns are fixed to ./internal/... and ./pkg/...; extra args may only be flags or flag values" >&2
            return 1
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
                return 1
                ;;
            -coverpkg=*)
                validate_coverpkg_value "${arg#-coverpkg=}" || return 1
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
        return 1
    fi
}

GO_ARGS=("test")
if [[ "$MODE" == "ci" ]]; then
    GO_ARGS+=("-short")
fi
if [[ "$MODE" == "dev" ]]; then
    COVERAGE_PROFILE_FILE="$ARTIFACT_DIR/coverage.out"
    COVERAGE_PROFILE_ARG="$(native_path "$COVERAGE_PROFILE_FILE")"
    GO_ARGS+=("-coverprofile=$COVERAGE_PROFILE_ARG")
fi
GO_ARGS+=("${EXTRA_ARGS[@]}" "${TARGETS[@]}")

write_command_file() {
    local prefix="$1"
    {
        if [[ -n "$prefix" ]]; then
            printf '%s' "$prefix"
        fi
        printf 'go'
        for arg in "${GO_ARGS[@]}"; do
            printf ' %q' "$arg"
        done
        printf '\n'
    } > "$ARTIFACT_DIR/command.txt"
}

if ! VALIDATION_OUTPUT="$(validate_extra_args "${EXTRA_ARGS[@]}" 2>&1)"; then
    EXIT_CODE=2
    write_command_file "validation failed before go test execution; blocked command: "
    if [[ -n "$VALIDATION_OUTPUT" ]]; then
        printf '%s\n' "$VALIDATION_OUTPUT" | tee "$ARTIFACT_DIR/stderr.log" >&2
    fi
    exit "$EXIT_CODE"
fi

write_command_file ""

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

if [[ "$MODE" == "dev" && -f "$COVERAGE_PROFILE_FILE" ]]; then
    "$GO_BIN" tool cover -func="$COVERAGE_PROFILE_ARG" | tee "$ARTIFACT_DIR/coverage-summary.txt"
fi

exit "$EXIT_CODE"
