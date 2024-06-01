
const API_PATH = "/bots"

export const PositionType_NIL    = 0
export const PositionType_E6AXIS = 1
export const PositionType_E6POS  = 2

export class Position {
  #valueType = ""
  #values = new Float32Array(14)

  constructor(valueType, values = undefined) {
    this.#valueType = valueType
    if (Boolean(values) === true) {
      this.Parse(values)
    }
  }

  Parse = (values) => {
    if (Array.isArray(values) === false) {
      throw new TypeError("Input type is not Array")
    }

    if (values.length !== 15) {
      throw new TypeError("Input incorrect length")
    }

    this.#valueType = values[0]

    for (let i = 0; i < 14; i++) {
      this.#values[i] = values[i + 1]
    }
  }

  #axes = () => `{E6AXIS: A1 ${this.#values[0].toFixed(5)}, A2 ${this.#values[1].toFixed(5)}, A3 ${this.#values[2].toFixed(5)}, A4 ${this.#values[3].toFixed(5)}, A5 ${this.#values[4].toFixed(5)}, A6 ${this.#values[5].toFixed(5)}}`

  #axesFull = () => `{E6AXIS: A1 ${this.#values[0].toFixed(5)}, A2 ${this.#values[1].toFixed(5)}, A3 ${this.#values[2].toFixed(5)}, A4 ${this.#values[3].toFixed(5)}, A5 ${this.#values[4].toFixed(5)}, A6 ${this.#values[5].toFixed(5)}, E1 ${this.#values[8].toFixed(5)}, E2 ${this.#values[9].toFixed(5)}, E3 ${this.#values[10].toFixed(5)}, E4 ${this.#values[11].toFixed(5)}, E5 ${this.#values[12].toFixed(5)}, E6 ${this.#values[13].toFixed(5)}}`

  #coords = () => `{E6POS: X  ${this.#values[0].toFixed(5)}, Y  ${this.#values[1].toFixed(5)}, Z  ${this.#values[2].toFixed(5)}, A  ${this.#values[3].toFixed(5)}, B  ${this.#values[4].toFixed(5)}, C  ${this.#values[5].toFixed(5)}}`

  #coordsFull = () => `{E6POS: X  ${this.#values[0].toFixed(5)}, Y  ${this.#values[1].toFixed(5)}, Z  ${this.#values[2].toFixed(5)}, A  ${this.#values[3].toFixed(5)}, B  ${this.#values[4].toFixed(5)}, C  ${this.#values[5].toFixed(5)}, S ${this.#values[6].toFixed(5)}, T ${this.#values[7].toFixed(5)}, E1 ${this.#values[8].toFixed(5)}, E2 ${this.#values[9].toFixed(5)}, E3 ${this.#values[10].toFixed(5)}, E4 ${this.#values[11].toFixed(5)}, E5 ${this.#values[12].toFixed(5)}, E6 ${this.#values[13].toFixed(5)}}`

  toString() {
    switch (this.#valueType) {
      case PositionType_E6AXIS:
        return this.#axesFull()
      case PositionType_E6POS:
        return this.#coordsFull()
    }
    return `[POSITION TYPE OF ${this.#valueType} ERROR]`
  }
}

export class MoveGroup {
  #id = ""
  get Id() { return this.#id } 

  #positions = []

  constructor(id, positions) {
    this.#id = id
    if (Array.isArray(positions)) {
      this.#positions = positions.map(value => new Position(PositionType_NIL, value))
    }
  }

  toString() {
    return this.#positions.reduce((acc, position) => acc + `<p>${position}</p>`, "")
  }

  move = async (bot) => {
    const body = new FormData()
    body.append("botId", String(bot) || "")
    body.append("moveGroupId", String(this.#id))
    try {
      const response = await fetch(API_PATH, { method: "POST", body })
      if (!response.ok) {
        throw new Error(`HTTP POST error! Status: ${response.status} Body: ${await response.body()}`)
      }
    } catch (error) {
      console.error(`[MoveGroup ERROR] Move error: ${error}`)
    }
  }
}
  