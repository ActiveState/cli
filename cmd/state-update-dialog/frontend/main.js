// Main entry point
function start() {
    el("initial-screen").style.display = "";

    let showContextMenu = document.oncontextmenu;
    document.oncontextmenu = () => false;
    backend.Bindings.DebugMode().then(debugMode => {
        console.log(debugMode);
        if (debugMode) document.oncontextmenu = showContextMenu; // Enable context menu if we're running in debug mode
    })

    Promise.all([
        backend.Bindings.CurrentVersion(),
        backend.Bindings.AvailableVersion()
    ]).then(result => {
        el("CurrentVersion").innerText = result[0];
        el("AvailableVersion").innerText = result[1];
    })

    backend.Bindings.Warning().then(result => {
        console.log(result);
        if (result === "") return;
        let projects = JSON.parse(result);
        if (projects.length === 0) return;

        for (let project of projects) {
            let li = document.createElement("li");
            li.textContent = project.Path;
            el("warning-projects").appendChild(li);
        }

        el("warning-wrapper").style.display = "block";
        el("warning").innerHTML = result;
    });

    populateChangelog();

    el("close-btn").addEventListener("click", () => backend.Bindings.Exit());
    el("close-btn2").addEventListener("click", () => backend.Bindings.Exit());
    el("install-btn").addEventListener("click", install);
}

function install() {
    el("install-btn").setAttribute("disabled", "true");
    el("close-btn2").setAttribute("disabled", "true");
    el("initial-screen").style.display = "none";
    el("install-screen").style.display = "";
    backend.Bindings.Install()
        .then(installProgress)
        .catch(installFailure);
}

function installFailure(message) {
    console.log("Failure: " + message);
    el("installerror-content").innerText = message;
}

function installProgress() {
    Promise.all([
        backend.Bindings.InstallReady(),
        backend.Bindings.InstallLog()
    ]).then(result => {
        let [installReady, installLog] = result;
        el("installog-content").innerText = installLog;
        if (!!installReady) {
            el("close-btn2").removeAttribute("disabled");
            return;
        }
        setTimeout(installProgress, 1000);
    }).catch(installFailure)
}

function populateChangelog(tries) {
    tries = tries || 0;
    if (tries > 10) {
        return;
    }
    backend.Bindings.Changelog().then(result => {
        if (result === "") {
            tries++;
            setTimeout(populateChangelog.bind(null, tries), tries * 100);
        } else {
            let changelog = el("changelog-content");
            changelog.innerHTML = result;
        }
    })
}

function el(id) {
    return document.getElementById(id);
}

// We provide our entrypoint as a callback for runtime.Init
window.wails._.Init(start);
