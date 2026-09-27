import { describe, expect, it } from 'vitest'
import { isFinal, nextStatuses } from './labels'

describe('nextStatuses', () => {
  it('segue o fluxo principal para o gestor', () => {
    expect(nextStatuses('aberta', 'gestor')).toEqual(['em_analise', 'cancelada'])
    expect(nextStatuses('em_analise', 'gestor')).toEqual(['em_atendimento', 'cancelada'])
    expect(nextStatuses('em_atendimento', 'gestor')).toEqual(['resolvida', 'cancelada'])
  })

  it('não permite transições a partir de estados finais', () => {
    expect(nextStatuses('resolvida', 'gestor')).toEqual([])
    expect(nextStatuses('cancelada', 'gestor')).toEqual([])
  })

  it('solicitante só cancela ocorrência aberta', () => {
    expect(nextStatuses('aberta', 'solicitante')).toEqual(['cancelada'])
    expect(nextStatuses('em_analise', 'solicitante')).toEqual([])
  })
})

describe('isFinal', () => {
  it('identifica estados finais', () => {
    expect(isFinal('resolvida')).toBe(true)
    expect(isFinal('cancelada')).toBe(true)
    expect(isFinal('aberta')).toBe(false)
  })
})
