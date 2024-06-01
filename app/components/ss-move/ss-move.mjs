
import { AbstractComponent } from "../abstract/index.mjs"

import CSS from "./ss-move.css" with { type: "css" }
const HTML = await (await fetch(import.meta.resolve("./ss-move.tpl"))).text()

export class MoveComponent extends AbstractComponent {

  #bot = undefined
	#moveGroup = undefined

	#idNode = this.$("#id")
	#positionSectionNode = this.$("#positionSection")
	#moveButtonNode = this.$("#moveButton")

	constructor(bot, moveGroup) {
    super(HTML, CSS)
    this.#bot = bot
    this.#moveGroup = moveGroup
  }

  #render = () => {
  	this.#idNode.innerText = this.#moveGroup.Id
  	this.#moveButtonNode.innerText = this.#moveGroup.Id
  	this.#positionSectionNode.innerHTML = `${this.#moveGroup}`
  }

  #moveClick = () => this.#moveGroup.move(this.#bot)

  connectedCallback() {
  	this.#render()
    this.#moveButtonNode.addEventListener("click", this.#moveClick)
  }

  disconnectedCallback() {
    this.#moveButtonNode.removeEventListener("click", this.#moveClick)
  }
}

customElements.define("ss-move", MoveComponent)