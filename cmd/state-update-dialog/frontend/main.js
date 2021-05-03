// Main entry point
function start() {
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
        document.getElementById("CurrentVersion").innerText = result[0];
        document.getElementById("AvailableVersion").innerText = result[1];
    })

    backend.Bindings.Warning().then(result => {
        if (result === "") return;
        document.getElementById("warning-wrapper").style.display = "block";
        document.getElementById("warning").innerHTML = result;
    });

    populateChangelog();

    document.getElementById("close-btn").addEventListener("click", () => backend.Bindings.Exit());
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
            let changelog = document.getElementById("changelog");
            changelog.style.height = "";
            changelog.innerHTML = result;
        }
    })
}

// We provide our entrypoint as a callback for runtime.Init
window.wails._.Init(start);
