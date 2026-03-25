export interface Seat {
  id: string
  seat_type: string
  title: string
  persona_id: string
  scope: string
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface BoardResponse {
  seat_type: string
  title: string
  persona_id: string
  content: string
}

export interface BoardDiscussResult {
  topic: string
  responses: BoardResponse[]
  synthesis: string
}
