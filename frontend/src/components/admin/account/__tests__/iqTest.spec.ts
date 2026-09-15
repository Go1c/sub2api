import { describe, expect, it } from 'vitest'
import { applyIqTestOutput, buildIqTestRequest, extractCompleteSvg, iqSvgToImageUrl } from '../iqTest'

const pelican = '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 80"><circle cx="50" cy="40" r="20"/></svg>'

describe('buildIqTestRequest', () => {
  it('sends mode=iq without extra handler fields', () => {
    expect(buildIqTestRequest('gpt-6-astra', 'Generate an SVG of a pelican riding a bicycle')).toEqual({
      model_id: 'gpt-6-astra',
      prompt: 'Generate an SVG of a pelican riding a bicycle',
      mode: 'iq'
    })
  })
})

describe('applyIqTestOutput', () => {
  it('returns a data url for a complete svg', () => {
    const url = applyIqTestOutput(`Sure.\n\n${pelican}`)
    expect(url).toContain('data:image/svg+xml')
    expect(decodeURIComponent(url || '')).toContain('<svg')
  })

  it('returns null until the svg is complete', () => {
    expect(applyIqTestOutput('<svg viewBox="0 0 10 10"><rect')).toBeNull()
  })
})

describe('iqSvgToImageUrl', () => {
  it('sanitizes a stored svg into an img data url', () => {
    const url = iqSvgToImageUrl(pelican)
    expect(url).toContain('data:image/svg+xml')
    expect(decodeURIComponent(url || '')).toContain('<svg')
  })

  it('returns null for empty svg', () => {
    expect(iqSvgToImageUrl('')).toBeNull()
  })
})

describe('extractCompleteSvg', () => {
  it('returns null for empty or non-svg text', () => {
    expect(extractCompleteSvg('')).toBeNull()
    expect(extractCompleteSvg('thinking about a pelican')).toBeNull()
  })

  it('extracts the svg after a thinking preamble', () => {
    const text = `Sure, I will draw that now.\n\n${pelican}\nDone.`
    expect(extractCompleteSvg(text)).toBe(pelican)
  })

  it('extracts svg from a markdown fence', () => {
    const text = `Here you go:\n\`\`\`svg\n${pelican}\n\`\`\`\n`
    expect(extractCompleteSvg(text)).toBe(pelican)
  })

  it('extracts svg from an xml fence', () => {
    const text = `\`\`\`xml\n${pelican}\n\`\`\``
    expect(extractCompleteSvg(text)).toBe(pelican)
  })

  it('does not render an incomplete svg', () => {
    expect(extractCompleteSvg('<svg viewBox="0 0 10 10"><rect')).toBeNull()
    expect(extractCompleteSvg('```svg\n<svg viewBox="0 0 10 10">\n```')).toBeNull()
  })

  it('takes the last complete svg when there are two', () => {
    const first = '<svg id="a"></svg>'
    const second = '<svg id="b"><circle cx="1" cy="1" r="1"/></svg>'
    expect(extractCompleteSvg(`first ${first}\nthen ${second}`)).toBe(second)
  })

  it('unescapes html-encoded svg', () => {
    const encoded = '&lt;svg xmlns="http://www.w3.org/2000/svg"&gt;&lt;circle cx="1" cy="1" r="1"/&gt;&lt;/svg&gt;'
    expect(extractCompleteSvg(`answer: ${encoded}`)).toBe(
      '<svg xmlns="http://www.w3.org/2000/svg"><circle cx="1" cy="1" r="1"/></svg>'
    )
  })

  it('strips a leading BOM', () => {
    expect(extractCompleteSvg(`\uFEFF${pelican}`)).toBe(pelican)
  })

  it('keeps the outer svg when nested', () => {
    const nested = '<svg id="outer"><svg id="inner"></svg></svg>'
    expect(extractCompleteSvg(nested)).toBe(nested)
  })
})
