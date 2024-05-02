package main

import (
    "maunium.net/go/mautrix/bridge"
    "mybridge/database"
)

type Portal struct {
    *database.Portal
}

func (br *MyBridge) GetAllIPortals() (iportals []bridge.Portal) {
    portals, err := br.dbPortalsToPortals(br.DB.Portal.GetAllWithMXID(context.TODO()))
    if err != nil {
        br.ZLog.Err(err).Msg("Failed to get all portals with mxid")
        return nil
    }
    iportals = make([]bridge.Portal, len(portals))
    for i, portal := range portals {
        iportals[i] = portal
    }
    return iportals
}


func (br *MyBridge) dbPortalsToPortals(dbPortals []*database.Portal, err error) ([]*Portal, error) {
	if err != nil {
		return nil, err
	}
	br.portalsLock.Lock()
	defer br.portalsLock.Unlock()

	output := make([]*Portal, len(dbPortals))
	for index, dbPortal := range dbPortals {
		if dbPortal == nil {
			continue
		}

		portal, ok := br.portalsByID[dbPortal.PortalKey]
		if !ok {
			portal = br.loadPortal(context.TODO(), dbPortal, nil)
		}

		output[index] = portal
	}

	return output, nil
}
