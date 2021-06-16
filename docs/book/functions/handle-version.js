function notFound(info) {
    return {
        statusCode: 404,
        headers: {'content-type': 'text/html'},
        body: ("<h1>Not Found</h1>"+
            "<p>You shouldn't see this page, please file a bug</p>"+
            `<details><summary>debug details</summary><pre><code>${JSON.stringify(info)}</code></pre></details>`
        ),
    };
}

function redirectToDownload(version, file) {
    const loc = `https://github.com/kubernetes-sigs/kubebuilder/releases/download/v${version}/${file}`;
    return {
        statusCode: 302,
        headers: {'location': loc, 'content-type': 'text/plain'},
        body: `Redirecting to ${loc}`,
    };
}


exports.handler = async function(evt, ctx) {
    // grab the prefix too to check for coherence
    const [prefix, version, os, arch] = evt.path.split("/").slice(-4);
    if (prefix !== 'releases' || !version || !os || !arch) {
        return notFound({version: version, os: os, arch: arch, prefix: prefix, rawPath: evt.path});
    }

    switch(version[0]) {
        case '1':
            // fallthrough
        case '2':
            return redirectToDownload(version, `kubebuilder_${version}_${os}_${arch}.tar.gz`);
        default:
            return redirectToDownload(version, `kubebuilder_${os}_${arch}`);
    }
}
