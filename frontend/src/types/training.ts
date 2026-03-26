export type TrainingProgramStatus = 'active' | 'archived'
export type EnrollmentStatus = 'enrolled' | 'in_progress' | 'completed' | 'dropped'

export interface TrainingProgram {
  id: string
  tenant_id: string
  title: string
  description: string
  category: string
  duration_hours: number
  provider: string
  url: string
  is_mandatory: boolean
  status: TrainingProgramStatus
  enrollment_count: number
  completed_count: number
  created_at: string
  updated_at: string
}

export interface TrainingEnrollment {
  id: string
  program_id: string
  employee_id: string
  status: EnrollmentStatus
  enrolled_at: string
  completed_at: string | null
  score: number | null
  notes: string
  employee_name?: string
  program_title?: string
  program_category?: string
}
