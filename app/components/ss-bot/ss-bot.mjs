
import { AbstractComponent } from "../abstract/index.mjs"
import { BotTeam } from "../../services/bot-team.mjs"

import { Position, PositionType_E6AXIS, PositionType_E6POS, MoveGroup } from "../../internals/position.mjs"
import { Template } from "../../internals/template.mjs"

import { MoveComponent } from "../ss-move/ss-move.mjs"

import CSS from "./ss-bot.css" with { type: "css" }
const HTML = await (await fetch(import.meta.resolve("./ss-bot.tpl"))).text()

export class BotComponent extends AbstractComponent  {
	
  #botTeam = undefined

  #MoveGroups = []

  #Name = ""
  get Name() { return this.#Name }

  #Address = ""
  get Address() { return this.#Address }

  #OSCRequestAxis = ""
  get OSCRequestAxis() { return this.#OSCRequestAxis }
  
  #OSCRequestCoords = ""
  get OSCRequestCoords() { return this.#OSCRequestCoords }

  #OSCRequestPosition = ""
  get OSCRequestPosition() { return this.#OSCRequestPosition }

  #OSCResponseAddress = ""
  get OSCResponseAddress() { return this.#OSCResponseAddress }

  #OSCResponseAxes = ""
  get OSCResponseAxes() { return this.#OSCResponseAxes }
  
  #OSCResponseCoords = ""
  get OSCResponseCoords() { return this.#OSCResponseCoords }
  
  #OSCResponsePosition = ""
  get OSCResponsePosition() { return this.#OSCResponsePosition }

  #TagId = ""
  get TagId() { return this.#TagId }

  #IsMovement = false
  get IsMovement() { return this.#IsMovement }

  #COM_ACTION = ""
  get COM_ACTION() { return this.#COM_ACTION }

  #COM_ROUNDM = ""
  get COM_ROUNDM() { return this.#COM_ROUNDM }

  #AXIS_ACT = new Position(PositionType_E6AXIS)
  get AXIS_ACT() { return this.#AXIS_ACT }
  
  #POS_ACT  = new Position(PositionType_E6POS)
  get POS_ACT() { return this.#POS_ACT }

  #OFFSET   = new Position(PositionType_E6POS)
  get OFFSET() { return this.#OFFSET }

  #POSITION = new Position(PositionType_E6POS)
  get POSITION() { return this.#POSITION }

  #PROXY_TYPE = ""
  get PROXY_TYPE() { return this.#PROXY_TYPE }

  #PROXY_VERSION = ""
  get PROXY_VERSION() { return this.#PROXY_VERSION }

  #PROXY_HOSTNAME = ""
  get PROXY_HOSTNAME() { return this.#PROXY_HOSTNAME }

  #PROXY_ADDRESS = ""
  get PROXY_ADDRESS() { return this.#PROXY_ADDRESS }

  #PROXY_PORT = ""
  get PROXY_PORT() { return this.#PROXY_PORT }

  #isDataReflected = false
  #isMoveReflected = false
  
  #infoTemplate = Template(this.$("#infoTemplate").innerHTML).bind(this)
  
  #infoSectionNode = this.$("#infoSection")
  #moveSectionNode = this.$("#moveSection")

  constructor() {
    super(HTML, CSS)
  }

  #render = () => {
    if (this.#isDataReflected) {
      return
    }
    this.#infoSectionNode.innerHTML = ""
    this.#infoSectionNode.appendChild(this.#infoTemplate())
    this.#isDataReflected = true

    if (this.#isMoveReflected) {
      return
    }

    const moveFragment = new DocumentFragment()
    this.#MoveGroups.forEach(moveGroup => moveFragment.appendChild(new MoveComponent(this.id, moveGroup)))
    this.#moveSectionNode.innerHTML = ""
    this.#moveSectionNode.appendChild(moveFragment)
    this.#isMoveReflected = true
  }

  #renderLoop = () => {
    this.#render()
    requestAnimationFrame(this.#renderLoop)
  }

  #getBotDataLoop = async id => {
    const botIterator = this.#botTeam.BotIterator(id)
    for await (const botData of botIterator) {
      this.#Name = botData.name
      this.#Address = botData.address

      this.#OSCRequestAxis = botData.oscRequestAxis
      this.#OSCRequestCoords = botData.oscRequestCoords
      this.#OSCRequestPosition = botData.oscRequestPosition
      
      this.#OSCResponseAddress = botData.oscResponseAddress

      this.#OSCResponseAxes = botData.oscResponseAxes
      this.#OSCResponseCoords = botData.oscResponseCoords
      this.#OSCResponsePosition = botData.oscResponsPosition

      this.#TagId = botData.tagID
      this.#IsMovement = botData.isMovement

      this.#COM_ACTION = botData.COM_ACTION
      this.#COM_ROUNDM = botData.COM_ROUNDM

      this.#AXIS_ACT = new Position(PositionType_E6AXIS, botData.AXIS_ACT)
      this.#POS_ACT  = new Position(PositionType_E6POS, botData.POS_ACT)
      this.#OFFSET   = new Position(PositionType_E6POS, botData.OFFSET)
      this.#POSITION = new Position(PositionType_E6POS, botData.POSITION)

      this.#PROXY_TYPE = botData.PROXY_TYPE
      this.#PROXY_VERSION = botData.PROXY_VERSION
      this.#PROXY_HOSTNAME = botData.PROXY_HOSTNAME
      this.#PROXY_ADDRESS = botData.PROXY_ADDRESS
      this.#PROXY_PORT = botData.PROXY_PORT

      this.#isDataReflected = false

      const length = this.#MoveGroups.length
      this.#MoveGroups = botData.moveGroups.map(({id, positions}) => new MoveGroup(id, positions))
      this.#isMoveReflected = this.#MoveGroups.length === length
    }
  }

  async connectedCallback() {
    this.#botTeam = await new BotTeam()
    const id = Number.parseInt(this.id)
    this.#getBotDataLoop(id)
        .then(error => console.error(`[BotTeam ERROR] Data loop then error: ${error}`))
        .catch(error => console.error(`[BotTeam ERROR] Data loop catch error: ${error}`))
    this.#renderLoop()
  }
}

customElements.define("ss-bot", BotComponent)
