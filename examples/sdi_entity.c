/*
 * Copyright (c) 2016 Dell Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you may
 * not use this file except in compliance with the License. You may obtain
 * a copy of the License at http://www.apache.org/licenses/LICENSE-2.0
 *
 * THIS CODE IS PROVIDED ON AN  *AS IS* BASIS, WITHOUT WARRANTIES OR
 * CONDITIONS OF ANY KIND, EITHER EXPRESS OR IMPLIED, INCLUDING WITHOUT
 *  LIMITATION ANY IMPLIED WARRANTIES OR CONDITIONS OF TITLE, FITNESS
 * FOR A PARTICULAR PURPOSE, MERCHANTABLITY OR NON-INFRINGEMENT.
 *
 * See the Apache Version 2.0 License for specific language governing
 * permissions and limitations under the License.
 */

/*
 * filename: sdi_entity.c
 */


/**************************************************************************************
 * Implementation of generic entity  and resource API.
 ***************************************************************************************/

#include "sdi_entity.h"

/**
 * Retrieve presence status of given entity.
 *
 * entity_hdl[in] - handle to the entity whose information has to be retrieved.
 * presence[out]    - true if entity is present, false otherwise
 *
 * return STD_ERR_OK on success and standard error on failure
 */
t_std_error sdi_entity_presence_get(sdi_entity_hdl_t entity_hdl, bool *presence)
{
    *presence = false;  /* Valid, but dummy value */

    return (STD_ERR_OK);
}

/**
 * Checks the fault status for a given entity
 *
 * entity_hdl[in] - handle to the entity whose information has to be retrieved.
 * fault[out] - true if entity has any fault, false otherwise.
 *
 * return STD_ERR_OK on success and standard error on failure
 */
t_std_error sdi_entity_fault_status_get(sdi_entity_hdl_t entity_hdl, bool *fault)
{
    *fault = false;  /* Valid, but dummy value */

    return (STD_ERR_OK);
}

/**
 * Checks the psu output power status for a given psu
 *
 * entity_hdl[in] - handle to the psu entity whose information has to be retrieved.
 * status[out] - true if psu output status is good , false otherwise.
 *
 * return STD_ERR_OK on success , standard error on failure
 */
t_std_error sdi_entity_psu_output_power_status_get(sdi_entity_hdl_t entity_hdl,
                                                   bool *status)
{
    *status = false;  /* Valid, but dummy value */

    return (STD_ERR_OK);
}
