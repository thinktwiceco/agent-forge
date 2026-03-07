(function (selector) {
  var el = document.querySelector(selector);
  if (!el) return;
  var proto =
    el.tagName === "TEXTAREA"
      ? window.HTMLTextAreaElement.prototype
      : window.HTMLInputElement.prototype;
  var setter = Object.getOwnPropertyDescriptor(proto, "value").set;
  setter.call(el, "");
  el.dispatchEvent(new Event("input", { bubbles: true }));
  el.dispatchEvent(new Event("change", { bubbles: true }));
})(%q);
