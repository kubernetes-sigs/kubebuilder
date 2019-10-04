#!/usr/bin/env bash
shopt -s extglob

cr_major_pattern=":warning:|$(printf "\xe2\x9a\xa0")"
cr_minor_pattern=":sparkles:|$(printf "\xe2\x9c\xa8")"
cr_patch_pattern=":bug:|$(printf "\xf0\x9f\x90\x9b")"
cr_docs_pattern=":book:|$(printf "\xf0\x9f\x93\x96")"
cr_other_pattern=":running:|$(printf "\xf0\x9f\x8f\x83")"

# cr::symbol-type-raw turns :xyz: and the corresponding emoji
# into one of "major", "minor", "patch", "docs", "other", or
# "unknown", ignoring the '!'
cr::symbol-type-raw() {
    case $1 in
        @(${cr_major_pattern})?('!'))
            echo "major"
            ;;
        @(${cr_minor_pattern})?('!'))
            echo "minor"
            ;;
        @(${cr_patch_pattern})?('!'))
            echo "patch"
            ;;
        @(${cr_docs_pattern})?('!'))
            echo "docs"
            ;;
        @(${cr_other_pattern})?('!'))
            echo "other"
            ;;
        *)
            echo "unknown"
            ;;
    esac
}

# cr::symbol-type turns :xyz: and the corresponding emoji
# into one of "major", "minor", "patch", "docs", "other", or
# "unknown".
cr::symbol-type() {
    local type_raw=$(cr::symbol-type-raw $1)
    if [[ ${type_raw} == "unknown" ]]; then
        echo "unknown"
        return
    fi

    if [[ $1 == *'!' ]]; then
        echo "major"
        return
    fi

    echo ${type_raw}
}

# git::is-release-branch-name checks if its argument is a release branch name
# (release-0.Y or release-X).
git::is-release-branch-name() {
    [[ ${1-must specify release branch name to check} =~ release-((0\.[[:digit:]])|[[:digit:]]+) ]]
}

# git::ensure-release-branch checks that we're on a release branch
git::ensure-release-branch() {
    local current_branch=$(git rev-parse --abbrev-ref HEAD)
    if ! git::is-release-branch-name ${current_branch}; then
        echo "branch ${current_branch} does not appear to be a release branch (release-X)" >&2
        exit 1
    fi
}

# git::export-current-version outputs the current version
# as exported variables (${maj,min,patch}_ver, last_tag) after
# checking that we're on the right release branch.
git::export-current-version() {
    # make sure we're on a release branch
    git::ensure-release-branch

    # deal with the release-0.1 branch, or similar
    local release_ver=${BASH_REMATCH[1]}
    maj_ver=${release_ver}
    local tag_pattern='v${maj_ver}.([[:digit:]]+).([[:digit]]+)'
    if [[ ${maj_ver} =~ 0\.([[:digit:]]+) ]]; then
        maj_ver=0
        min_ver=${BASH_REMATCH[1]}
        local tag_pattern="v0.(${min_ver}).([[:digit:]]+)"
    fi

    # make sure we've got a tag that matches our release branch
    last_tag=$(git describe --tags --abbrev=0) # try to fetch just the "current" tag name
    if [[ ! ${last_tag} =~ ${tag_pattern} ]]; then
        echo "tag ${last_tag} does not appear to be a release for this release (${release_ver})-- it should be v${maj_ver}.Y.Z" >&2
        exit 1
    fi

    export min_ver=${BASH_REMATCH[1]}
    export patch_ver=${BASH_REMATCH[2]}
    export maj_ver=${maj_ver}
    export last_tag=${last_tag}
}

# git::next-version figures out the next version to tag
# (it also sets the current version variables to the current version)
git::next-version() {
    git::export-current-version

    local feature_commits=$(git rev-list ${last_tag}..${end_range} --grep="${cr_minor_pattern}")
    local breaking_commits=$(git rev-list ${last_tag}..${end_range} --grep="${cr_major_pattern}")

    if [[ -z ${breaking_commits} && ${maj_ver} > 0 ]]; then
        local next_ver="v$(( maj_ver + 1 )).0.0"
    elif [[ -z ${feature_commits} ]]; then
        local next_ver="v${maj_ver}.$(( min_ver + 1 )).0"
    else
        local next_ver="v${maj_ver}.${min_ver}.$(( patch_ver + 1 ))"
    fi

    echo "${next_ver}"
}
