import {expect, test} from '@playwright/test'
import {randomUUID} from 'node:crypto'
import {account, session} from './accounts'

test('registration feeds enrollment and a retried placement creates one row', async ({request}) => {
  const headers = await session(request, account('A'))
  const created = await request.post('/api/v1/students', {
    headers, data: {givenName: 'Sam', familyName: 'Placement ' + randomUUID().slice(0, 8)}
  })
  expect(created.status()).toBe(201)
  const student = await created.json()
  const url = '/api/v1/enrollment/placements'
  const placement = {studentId: student.id, academicYear: '2026-2027', grade: 'CP'}
  try {
    const first = await request.post(url, {headers, data: placement})
    expect(first.status()).toBe(201)
    const saved = await first.json()
    const replay = await request.post(url, {headers, data: placement})
    expect(replay.status()).toBe(200)
    expect((await replay.json()).id).toBe(saved.id)
    const list = await request.get(url, {headers, params: {studentId: student.id}})
    expect((await list.json()).total).toBe(1)
    const stranger = await session(request, account('B'))
    expect((await request.post(url, {headers: stranger, data: placement})).status()).toBe(404)
    const withdraw = await request.patch(url + '/' + saved.id, {
      headers, data: {version: saved.version, status: 'withdrawn'}
    })
    expect(withdraw.status()).toBe(200)
    expect((await withdraw.json()).status).toBe('withdrawn')
  } finally {
    const archive = await request.delete('/api/v1/students/' + student.id, {headers, params: {version: student.version}})
    expect(archive.status()).toBe(204)
  }
})

test('a document round trip preserves bytes and another school cannot download it', async ({request}) => {
  const owner = await session(request, account('A'))
  const other = await session(request, account('B'))
  const data = Buffer.from('Réunion pédagogique – ' + randomUUID(), 'utf8')
  const response = await request.post('/api/v1/documents/files', {
    headers: {...owner, 'Content-Type': 'text/plain'}, data
  })
  expect(response.status()).toBe(201)
  const {id} = await response.json()
  const url = '/api/v1/documents/files/' + id
  const download = await request.get(url, {headers: owner})
  expect(download.status()).toBe(200)
  expect(download.headers()['content-disposition']).toContain('attachment')
  expect(await download.body()).toEqual(data)
  expect((await request.get(url, {headers: other})).status()).toBe(404)
})
