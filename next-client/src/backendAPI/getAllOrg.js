const apiURL = process.env.APIURL ? process.env.APIURL : `http://localhost:8080`

export const getAllOrg = async () => {
    const res = await fetch(`${apiURL}/organization`, { cache: 'no-store' })
    const orgs = await res.json();
    console.log(orgs)
    return orgs
}

export const getOrg = async (id) => {
    const res = await fetch(`${apiURL}/organization/${id}`, { cache: 'no-store' })
    if (res.status == 200) {
        const org = await res.json();
        return org
    } else {
        return
    }
}

export const createSub = async (id) => {

}
export const cancelSub = async (id) => {
    try {
        const res = await fetch(`${apiURL}/organization/${id}/sub`, { method: 'delete', cache: 'no-store' })
        if (res.status == 200) {
            return
        } else {
            return
        }
    }catch(err) {
        console.log(err)
    }

}

export const getSub = async(id) => {
    const res = await fetch(`${apiURL}/organization/${id}/sub`, { cache: 'no-store' })
    if (res.status == 200) {
        return await res.json()
    } else {
        return
    }
}

export const getPlans = async() => {
    const res = await fetch(`${apiURL}/plans`, { cache: 'no-store' })
    if (res.status == 200) {
        return await res.json()
    } else {
        return
    }
}