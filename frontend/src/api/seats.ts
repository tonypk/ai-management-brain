import { get, post } from './client'
import type { Seat, BoardDiscussResult } from '@/types'

export async function listSeats(): Promise<Seat[]> {
  const res = await get<{ data: Seat[] }>('/seats')
  return res.data
}

export async function boardDiscuss(topic: string): Promise<BoardDiscussResult> {
  const res = await post<{ data: BoardDiscussResult }>('/board/discuss', { topic })
  return res.data
}

export interface SeatChatResponse {
  seat_type: string
  content: string
}

export async function chatWithSeat(seatType: string, message: string): Promise<SeatChatResponse> {
  const res = await post<{ data: SeatChatResponse }>('/seats/chat', { seat_type: seatType, message })
  return res.data
}
