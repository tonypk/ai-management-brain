import { get, put } from './client'
import type { MentorConfig, MentorWithDomain } from '@/types'

export async function listMentors(): Promise<MentorWithDomain[]> {
  const res = await get<{ data: MentorWithDomain[] }>('/mentors')
  return res.data
}

export async function getMentorConfig(): Promise<MentorConfig> {
  const res = await get<{ data: MentorConfig }>('/mentor')
  return res.data
}

export async function switchMentor(mentorId: string): Promise<void> {
  await put<{ data: unknown }>('/mentor', { mentor_id: mentorId })
}
