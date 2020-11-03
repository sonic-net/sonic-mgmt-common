////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2020 Dell, Inc.                                                 //
//                                                                            //
//  Licensed under the Apache License, Version 2.0 (the "License");           //
//  you may not use this file except in compliance with the License.          //
//  You may obtain a copy of the License at                                   //
//                                                                            //
//  http://www.apache.org/licenses/LICENSE-2.0                                //
//                                                                            //
//  Unless required by applicable law or agreed to in writing, software       //
//  distributed under the License is distributed on an "AS IS" BASIS,         //
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  //
//  See the License for the specific language governing permissions and       //
//  limitations under the License.                                            //
//                                                                            //
////////////////////////////////////////////////////////////////////////////////

package transformer

import (
	"github.com/Azure/sonic-mgmt-common/translib/db"
    log "github.com/golang/glog"
)

func xfmrHandlerFunc(inParams XfmrParams, xfmrFuncNm string) (error) {
	xfmrLogInfoAll("Received inParams %v Subtree function name %v", inParams, xfmrFuncNm)
	if inParams.uri != inParams.requestUri {
		_, yerr := xlateUnMarshallUri(inParams.ygRoot, inParams.uri)
		if yerr != nil {
			xfmrLogInfoAll("Failed to generate the ygot Node for uri(\"%v\") err(%v).", inParams.uri, yerr)
		}
	}
	ret, err := XlateFuncCall(dbToYangXfmrFunc(xfmrFuncNm), inParams)
	if err != nil {
		xfmrLogInfoAll("Failed to retrieve data for xpath(\"%v\") err(%v).", inParams.uri, err)
		return err
	}
        if ((ret != nil) && (len(ret)>0)) {
		// db to yang subtree xfmr returns err as the only value in return data list from <xfmr_func>.Call()
		if ret[DBTY_SBT_XFMR_RET_ERR_INDX].Interface() != nil {
			err = ret[DBTY_SBT_XFMR_RET_ERR_INDX].Interface().(error)
			if err != nil {
				log.Warningf("Transformer function(\"%v\") returned error - %v.", xfmrFuncNm, err)
			}
		}
        }
	return err
}

func leafXfmrHandlerFunc(inParams XfmrParams, xfmrFieldFuncNm string) (map[string]interface{}, error) {
	var err error
	var fldValMap map[string]interface{}

	xfmrLogInfoAll("Received inParams %v to invoke Field transformer %v", inParams, xfmrFieldFuncNm)
	ret, err := XlateFuncCall(dbToYangXfmrFunc(xfmrFieldFuncNm), inParams)
	if err != nil {
		return fldValMap, err
	}
	if ((ret != nil) && (len(ret)>0)) {
		if len(ret) == DBTY_FLD_XFMR_RET_ARGS {
			// field xfmr returns err as second value in return data list from <xfmr_func>.Call()
                        if ret[DBTY_FLD_XFMR_RET_ERR_INDX].Interface() != nil {
                                err = ret[DBTY_FLD_XFMR_RET_ERR_INDX].Interface().(error)
                                if err != nil {
                                        log.Warningf("Transformer function(\"%v\") returned error - %v.", xfmrFieldFuncNm, err)
                                }
                        }
                }

                if ret[DBTY_FLD_XFMR_RET_VAL_INDX].Interface() != nil {
                        fldValMap = ret[DBTY_FLD_XFMR_RET_VAL_INDX].Interface().(map[string]interface{})
                }
        }
	return fldValMap, err
}

func keyXfmrHandlerFunc(inParams XfmrParams, xfmrFuncNm string) (map[string]interface{}, error) {
        xfmrLogInfoAll("Received inParams %v key transformer function name %v", inParams, xfmrFuncNm)
        ret, err := XlateFuncCall(dbToYangXfmrFunc(xfmrFuncNm), inParams)
        retVal := make(map[string]interface{})
        if err != nil {
                return retVal, err
        }

        if ((ret != nil) && (len(ret)>0)) {
                if len(ret) == DBTY_KEY_XFMR_RET_ARGS {
                        // key xfmr returns err as second value in return data list from <xfmr_func>.Call()
                        if ret[DBTY_KEY_XFMR_RET_ERR_INDX].Interface() != nil {
                                err = ret[DBTY_KEY_XFMR_RET_ERR_INDX].Interface().(error)
                                if err != nil {
                                        log.Warningf("Transformer function(\"%v\") returned error - %v.", xfmrFuncNm, err)
                                        return retVal, err
                                }
                        }
                }
                if ret[DBTY_KEY_XFMR_RET_VAL_INDX].Interface() != nil {
                        retVal = ret[DBTY_KEY_XFMR_RET_VAL_INDX].Interface().(map[string]interface{})
                        return retVal, nil
                }
        }
        return retVal, nil
}

func validateHandlerFunc(inParams XfmrParams, validateFuncNm string) (bool) {
	xfmrLogInfoAll("Received inParams %v, validate transformer function name %v", inParams, validateFuncNm)
	ret, err := XlateFuncCall(validateFuncNm, inParams)
	if err != nil {
		return false
	}
	return ret[0].Interface().(bool)
}

func xfmrTblHandlerFunc(xfmrTblFunc string, inParams XfmrParams, xfmrTblKeyCache map[string]tblKeyCache) ([]string, error) {

	xfmrLogInfoAll("Received inParams %v, table transformer function name %v", inParams, xfmrTblFunc)
	if (inParams.oper == GET && xfmrTblKeyCache != nil) {
		if tkCache, _ok := xfmrTblKeyCache[inParams.uri]; _ok {
			if len(tkCache.dbTblList) > 0 {
				return tkCache.dbTblList, nil
			}
		}
	}

	var retTblLst []string
	ret, err := XlateFuncCall(xfmrTblFunc, inParams)
	if err != nil {
		return retTblLst, err
	}
	if ((ret != nil) && (len(ret)>0)) {
		if len(ret) == TBL_XFMR_RET_ARGS {
			// table xfmr returns err as second value in return data list from <xfmr_func>.Call()
			if ret[TBL_XFMR_RET_ERR_INDX].Interface() != nil {
				err = ret[TBL_XFMR_RET_ERR_INDX].Interface().(error)
				if err != nil {
					log.Warningf("Transformer function(\"%v\") returned error - %v.", xfmrTblFunc, err)
					return retTblLst, err
				}
			}
		}

		if ret[TBL_XFMR_RET_VAL_INDX].Interface() != nil {
			retTblLst = ret[TBL_XFMR_RET_VAL_INDX].Interface().([]string)
		}
	}
	if (inParams.oper == GET && xfmrTblKeyCache != nil) {
		if _, _ok := xfmrTblKeyCache[inParams.uri]; !_ok {
			xfmrTblKeyCache[inParams.uri] = tblKeyCache{}
		}
		tkCache := xfmrTblKeyCache[inParams.uri]
		tkCache.dbTblList = retTblLst
		xfmrTblKeyCache[inParams.uri] = tkCache
	}

	return retTblLst, err
}

func valueXfmrHandler(inParams XfmrDbParams, xfmrValueFuncNm string) (string, error) {
	xfmrLogInfoAll("Received inParams %v Value transformer name %v", inParams, xfmrValueFuncNm)

	ret, err := XlateFuncCall(xfmrValueFuncNm, inParams)
	if err != nil {
		return "", err
	}

	if ((ret != nil) && (len(ret)>0)) {
		if len(ret) == YTDB_FLD_XFMR_RET_ARGS {
			// value xfmr returns err as second value in return data list from <xfmr_func>.Call()
			if ret[YTDB_FLD_XFMR_RET_ERR_INDX].Interface() != nil {
				err = ret[YTDB_FLD_XFMR_RET_ERR_INDX].Interface().(error)
				if err != nil {
					log.Warningf("Transformer function(\"%v\") returned error - %v.", xfmrValueFuncNm, err)
					return "", err
				}
			}
		}

		if ret[YTDB_FLD_XFMR_RET_VAL_INDX].Interface() != nil {
			retVal := ret[YTDB_FLD_XFMR_RET_VAL_INDX].Interface().(string)
			return retVal, nil
		}
	}

	return "", err
}

func leafXfmrHandler(inParams XfmrParams, xfmrFieldFuncNm string) (map[string]string, error) {
	xfmrLogInfoAll("Received inParams %v Field transformer name %v", inParams, xfmrFieldFuncNm)
	ret, err := XlateFuncCall(yangToDbXfmrFunc(xfmrFieldFuncNm), inParams)
	if err != nil {
		return nil, err
	}
	if ((ret != nil) && (len(ret)>0)) {
		if len(ret) == YTDB_FLD_XFMR_RET_ARGS {
			// field xfmr returns err as second value in return data list from <xfmr_func>.Call()
			if ret[YTDB_FLD_XFMR_RET_ERR_INDX].Interface() != nil {
				err = ret[YTDB_FLD_XFMR_RET_ERR_INDX].Interface().(error)
				if err != nil {
					log.Warningf("Transformer function(\"%v\") returned error - %v.", xfmrFieldFuncNm, err)
					return nil, err
				}
			}
		}

		if ret[YTDB_FLD_XFMR_RET_VAL_INDX].Interface() != nil {
			fldValMap := ret[YTDB_FLD_XFMR_RET_VAL_INDX].Interface().(map[string]string)
			return fldValMap, nil
		}
	} else {
		retFldValMap := map[string]string{"NULL":"NULL"}
		return retFldValMap, nil
	}

	return nil, nil
}

func xfmrHandler(inParams XfmrParams, xfmrFuncNm string) (map[string]map[string]db.Value, error) {
	xfmrLogInfoAll("Received inParams %v Subtree function name %v", inParams, xfmrFuncNm)
	ret, err := XlateFuncCall(yangToDbXfmrFunc(xfmrFuncNm), inParams)
	if err != nil {
		return nil, err
	}

	if ((ret != nil) && (len(ret)>0)) {
		if len(ret) == YTDB_SBT_XFMR_RET_ARGS {
			// subtree xfmr returns err as second value in return data list from <xfmr_func>.Call()
			if ret[YTDB_SBT_XFMR_RET_ERR_INDX].Interface() != nil {
				err = ret[YTDB_SBT_XFMR_RET_ERR_INDX].Interface().(error)
				if err != nil {
					log.Warningf("Transformer function(\"%v\") returned error - %v.", xfmrFuncNm, err)
					return nil, err
				}
			}
		}
		if ret[YTDB_SBT_XFMR_RET_VAL_INDX].Interface() != nil {
			retMap := ret[YTDB_SBT_XFMR_RET_VAL_INDX].Interface().(map[string]map[string]db.Value)
			return retMap, nil
		}
	}
	return nil, nil
}

func keyXfmrHandler(inParams XfmrParams, xfmrFuncNm string) (string, error) {
	xfmrLogInfoAll("Received inParams %v key transformer function name %v", inParams, xfmrFuncNm)
	ret, err := XlateFuncCall(yangToDbXfmrFunc(xfmrFuncNm), inParams)
	retVal := ""
	if err != nil {
		return retVal, err
	}

	if ((ret != nil) && (len(ret)>0)) {
		if len(ret) == YTDB_KEY_XFMR_RET_ARGS {
			// key xfmr returns err as second value in return data list from <xfmr_func>.Call()
			if ret[YTDB_KEY_XFMR_RET_ERR_INDX].Interface() != nil {
				err = ret[YTDB_KEY_XFMR_RET_ERR_INDX].Interface().(error)
				if err != nil {
					log.Warningf("Transformer function(\"%v\") returned error - %v.", xfmrFuncNm, err)
					return retVal, err
				}
			}
		}
		if ret[YTDB_KEY_XFMR_RET_VAL_INDX].Interface() != nil {
			retVal = ret[YTDB_KEY_XFMR_RET_VAL_INDX].Interface().(string)
			return retVal, nil
		}
	}
	return retVal, nil
}

/* Invoke the post tansformer */
func postXfmrHandlerFunc(xfmrPost string, inParams XfmrParams) (map[string]map[string]db.Value, error) {
	retData := make(map[string]map[string]db.Value)
	xfmrLogInfoAll("Received inParams %v, post transformer function name %v", inParams, xfmrPost)
	ret, err := XlateFuncCall(xfmrPost, inParams)
	if err != nil {
		return nil, err
	}
	if ((ret != nil) && (len(ret)>0)) {
		if len(ret) == POST_XFMR_RET_ARGS {
			// post xfmr returns err as second value in return data list from <xfmr_func>.Call()
			if ret[POST_XFMR_RET_ERR_INDX].Interface() != nil {
				err = ret[POST_XFMR_RET_ERR_INDX].Interface().(error)
				if err != nil {
					log.Warningf("Transformer function(\"%v\") returned error - %v.", xfmrPost, err)
					return retData, err
				}
			}
		}
		if ret[POST_XFMR_RET_VAL_INDX].Interface() != nil {
			retData = ret[POST_XFMR_RET_VAL_INDX].Interface().(map[string]map[string]db.Value)
			xfmrLogInfoAll("Post Transformer function : %v retData : %v", xfmrPost, retData)
		}
	}
	return retData, err
}

/* Invoke the pre tansformer */
func preXfmrHandlerFunc(xfmrPre string, inParams XfmrParams) error {
	xfmrLogInfoAll("Received inParams %v, pre transformer function name %v", inParams, xfmrPre)
	ret, err := XlateFuncCall(xfmrPre, inParams)
	if err != nil {
		log.Warningf("Pre-transformer function(\"%v\") returned error - %v.", xfmrPre, err)
		return err
	}
	if ((ret != nil) && (len(ret)>0)) {
		if ret[PRE_XFMR_RET_ERR_INDX].Interface() != nil {
			err = ret[PRE_XFMR_RET_ERR_INDX].Interface().(error)
			if err != nil {
				log.Warningf("Transformer function(\"%v\") returned error - %v.", xfmrPre, err)
				return err
			}
		}
	}
	return err
}

