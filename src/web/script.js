const endPtPane = document.getElementById("endPtPane");

async function openEndPt(elm, event) {
  elmClass = elm.className;
  try {
    const resp = await fetch(`/endpoints/${elmClass}.elh`);
    if (!resp.ok) {
      throw new Error(`Failed to fetch endpoint ${resp.Status}`);
    }

    const endPtHTML = await resp.text();
    const parser = new DOMParser();
    const endPtDOM = parser.parseFromString(endPtHTML, "text/html");
    const endPtScript = document.createElement("script");
    endPtScript.textContent = endPtDOM.querySelector("script").innerText;
    endPtScript.class = "parsed";
    endPtDOM.querySelector("script:not(.parsed)").remove();
    const elms = [
        endPtDOM.querySelector(".ptName"),
        endPtDOM.querySelector(".pt"),
        endPtScript,
    ];
    endPtPane.innerHTML = "";
    for (i = 0; i < elms.length; i++) {
      endPtPane.appendChild(elms[i]);
    }
  } catch (err) {
    console.error(err);
    endPtPane.innerHTML = `<h1 style="color: red">`+
          `err, failed to fetch endpoint: ${err}`;
  }
}
