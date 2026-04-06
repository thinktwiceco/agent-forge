(function (selector, requireVisible) {
  function findDeep(root, sel) {
    if (!root) return null;
    try {
      if (root.nodeType === 1 && root.matches && root.matches(sel)) return root;
    } catch (e) {}
    if (root.querySelector) {
      var el = root.querySelector(sel);
      if (el) return el;
    }
    var kids = root.children || root.childNodes;
    if (!kids) return null;
    for (var i = 0; i < kids.length; i++) {
      var ch = kids[i];
      if (ch.nodeType !== 1) continue;
      if (ch.shadowRoot) {
        var f = findDeep(ch.shadowRoot, sel);
        if (f) return f;
      }
      f = findDeep(ch, sel);
      if (f) return f;
    }
    return null;
  }

  function findEverywhere(doc, sel) {
    var el = findDeep(doc, sel);
    if (el) return el;
    var iframes = doc.querySelectorAll("iframe");
    for (var j = 0; j < iframes.length; j++) {
      try {
        var idoc = iframes[j].contentDocument;
        if (idoc) {
          var g = findEverywhere(idoc, sel);
          if (g) return g;
        }
      } catch (e) {}
    }
    return null;
  }

  function visible(el) {
    if (!el) return false;
    var win = el.ownerDocument.defaultView || window;
    var r = el.getBoundingClientRect();
    if (r.width < 1 || r.height < 1) return false;
    var st = win.getComputedStyle(el);
    if (st.visibility === "hidden" || st.display === "none") return false;
    if (parseFloat(st.opacity) === 0) return false;
    return true;
  }

  var el = findEverywhere(document, selector);
  if (!el) return { ok: false, reason: "not_found" };
  if (requireVisible && !visible(el)) return { ok: false, reason: "not_visible" };

  var win = el.ownerDocument.defaultView || window;
  win.focus && win.focus();
  el.scrollIntoView({ block: "center", inline: "nearest", behavior: "instant" });

  try {
    el.click();
  } catch (e) {
    return { ok: false, reason: "click_error" };
  }
  return { ok: true, reason: "" };
})(%q, %t);
