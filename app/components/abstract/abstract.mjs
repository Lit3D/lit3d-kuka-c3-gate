
export class AbstractComponent extends HTMLElement {
  constructor(HTML = "", CSS) {
    super()
    const root = this.attachShadow({mode: "open"})
    root.innerHTML = HTML
    if (CSS) root.adoptedStyleSheets = [CSS]
  }

  $ = selector => this.shadowRoot.querySelector(selector)
}
