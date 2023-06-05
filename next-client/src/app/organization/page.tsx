import AddList from "../components/AddList"
import GetList from "../components/GetList"
import { getAllOrg } from "../../backendAPI/getAllOrg"
export default async function Home() {
  const allOrg = await getAllOrg()

  return (
      <div className="text-center my-5 flex flex-col gap-4">
          <h1 className="font-extrabold text-transparent text-6xl bg-clip-text bg-gradient-to-r from-purple-400 to-pink-600" >
            List of Organization
          </h1>
          <AddList/>
          <GetList allOrg={allOrg}/>
      </div>
  )
}
