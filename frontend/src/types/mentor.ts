export interface MentorConfig {
  current_mentor_id: string
  current_blend: BlendConfig | null
  available_mentors: MentorInfo[]
}

export interface BlendConfig {
  primary_id: string
  secondary_id: string
  weight: number
}

export interface MentorInfo {
  id: string
  name: string
  description: string
}

export interface MentorWithDomain {
  id: string
  name: string
  name_en: string
  company: string
  philosophy: string
  domain: string
  tags: string[]
  recommended_seats: string[]
}
