#!/usr/bin/env bash

set -e
set -o pipefail

# import our common stuff
source "$(dirname ${BASH_SOURCE})/common.sh"

# TODO: work with both release branch and major release
git::ensure-release-branch
git::export-current-version
# check the next version
next_ver=$(git::next-version)

features=""
bugfixes=""
breaking=""
unknown=""
MERGE_PR="Merge pull request #([[:digit:]]+) from ([[:alnum:]-]+)/.+"
NEWLINE="
"
head_commit=$(git rev-parse HEAD)
while read commit_word commit; do
    read title
    read # skip the blank line
    read prefix body

    if [[ ${prefix} == v*.*.* && ( ${commit} == ${head_commit} || $(git tag --points-at ${commit}) == v*.*.* ) ]]; then
        # skip version merges
        continue
    fi
    set +x
    if [[ ! ${title} =~ ${MERGE_PR} ]]; then
        echo "Unable to determine PR number for merge ${commit} with title '${title}', aborting." >&2
        exit 1
    fi
    pr_number=${BASH_REMATCH[1]}
    pr_type=$(cr::symbol-type ${prefix})
    pr_title=${body}
    if [[ ${pr_type} == "unknown" ]]; then
        pr_title="${prefix} ${pr_title}"
    fi
    case ${pr_type} in
        major)
            breaking="${breaking}- ${pr_title} (#${pr_number})${NEWLINE}"
            ;;
        minor)
            features="${features}- ${pr_title} (#${pr_number})${NEWLINE}"
            ;;
        patch)
            bugfixes="${bugfixes}- ${pr_title} (#${pr_number})${NEWLINE}"
            ;;
        docs|other)
            # skip non-code-changes
            ;;
        unknown)
            unknown="${unknown}- ${pr_title} (#${pr_number})${NEWLINE}"
            ;;
        *)
            echo "unknown PR type '${pr_type}' on PR '${pr_title}'" >&2
            exit 1
    esac
done <<<$(git rev-list ${last_tag}..HEAD --merges --pretty=format:%B)

# TODO: sort non merge commits with tags

[[ -n "${breaking}" ]] && printf '\e[1;31mbreaking changes this version\e[0m' >&2
[[ -n "${unknown}" ]] && printf '\e[1;35munknown changes in this release -- categorize manually\e[0m' >&2

echo "" >&2
echo "" >&2
echo "# ${next_ver}"

if [[ -n ${breaking} ]]; then
    echo ""
    echo "## :warning: Breaking Changes"
    echo ""
    echo "${breaking}"
fi

if [[ -n ${features} ]]; then
    echo ""
    echo "## :sparkles: New Features"
    echo ""
    echo "${features}"
fi

if [[ -n ${bugfixes} ]]; then
    echo ""
    echo "## :bug: Bug Fixes"
    echo ""
    echo "${bugfixes}"
fi

if [[ -n ${unknown} ]]; then
    echo ""
    echo "## :question: *categorize these manually*"
    echo ""
    echo "${unknown}"
fi

echo ""
echo "*Thanks to all our contributors!*"
