
import { TOrg } from '@/interface/all'
import { getOrg } from '../../../backendAPI/getAllOrg'
import View from '../../components/View'
export default async function Organization({ params }: { params : { orgID: string} }) {
  const org = await getOrg(params.orgID)
  return (
    <>
      <h1 className="font-extrabold text-transparent text-6xl bg-clip-text bg-gradient-to-r from-purple-400 to-pink-600" >
        {params.orgID}
      </h1>
      <View org={(org as TOrg)}/>
    </>
  )
}
