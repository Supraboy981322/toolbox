ptItems = document.querySelectorAll(".itm").length;
for (i = 0; i < ptItems.length; i++) {
  elm = ptItems[i];
  pt = elm.querySelector(".pt"); 
  pt.addEventListener('click', (event) => {
    event.stopPropagation();
  });
}

function dropDown(elm, event) {
  elmClass = event.target.className; 
  if (elmClass != "ptName" && elmClass != "itm") {
    return;
  }
  ptDiv = elm.querySelector(".pt");
  if (elm.getAttribute("open") == "true") {
    elm.setAttribute("open", "false");
  } else {
    elm.setAttribute("open", "true");
  }
}
