import { NextResponse } from "next/server";


export async function GET(request: Request) {
    const resp = {
        'name' : 'Admin',
        'email' : 'admin@example.com',
    }
    return  NextResponse.json(resp);
  }
