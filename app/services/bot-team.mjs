
const API_PATH = "/bots"
const API_REQUEST_TIMEOUT = 10 // 1 * 1000
const API_ERROR_TIMEOUT = 5 * 1000

class BotIterator {
  #messageQueue = []
  #nextResolve = undefined

  Next = (value = undefined) => {
    if (this.#nextResolve) {
      this.#nextResolve({ value, done: value === null })
      this.#nextResolve = undefined
      return
    }
    this.#messageQueue.push(value)
  }

  [Symbol.asyncIterator] = () => {
    return {
      next: () => {
        if (this.#messageQueue.length > 0) {
          const value = this.#messageQueue.shift()
          return Promise.resolve({ value, done: value === null })
        }
        return new Promise((resolve) => this.#nextResolve = resolve)
      }
    }
  }
}

export class BotTeam {
	static #instance = undefined

  #botTeam = []
  #botIterators = new Map()

	constructor() {
    return (BotTeam.#instance = BotTeam.#instance ?? this.#init())
  }

  #init = async (offer) => {
    this.#loop()
        .catch(error => console.error(`[BotTeam ERROR] Loop catch error: ${error}`))
        .then(error => console.error(`[BotTeam ERROR] Loop then error: ${error}`))
    return this
  }

  #loop = async () => {
    while (true) {
      await new Promise(resolve => setTimeout(resolve, API_REQUEST_TIMEOUT))
      try {
        const response = await fetch(API_PATH)
        if (!response.ok) {
          throw new Error(`HTTP GET error! Status: ${response.status} Body: ${await response.body()}`)
        }
        this.#botTeam = await response.json()
      } catch (error) {
        console.error(`[BotTeam ERROR] Loop error: ${error}`)
        await new Promise(resolve => setTimeout(resolve, API_ERROR_TIMEOUT))
      }

      this.#botIterators.forEach((botIterator, id) => {
        const value = this.#botTeam[id]
        if (typeof value == "object") {
          botIterator.Next(value)
        } 
      })
    }
  }

  BotIterator = id => {
    const botIterator = new BotIterator()
    this.#botIterators.set(id, botIterator)
    return botIterator
  }
}
