
export const Template = (tpl, object) => new Function(`
renderer = document.createElement("template")
renderer.innerHTML = \`${tpl.replaceAll("`","&DiacriticalGrave;")}\`                             
return renderer.content
`)